package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AcronymEntry represents one row in the CSV
type AcronymEntry struct {
	Full        string `json:"full" jsonschema:"the full form of the acronym"`
	Description string `json:"description" jsonschema:"a description of the acronym"`
}

type LookupAcronymInput struct {
	Acronym string `json:"acronym" jsonschema:"The acronym or initialism to resolve."`
}

type LookupAcronymOutput struct {
	Acronym     string         `json:"acronym" jsonschema:"the sanitized acronym key"`
	Definitions []AcronymEntry `json:"definitions" jsonschema:"matching definitions"`
}

var nonAlpha = regexp.MustCompile("[^A-Za-z]+")

var Version = "dev" // This will be set by the build systems to the release version

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

func buildVersionOutput(version string) string {
	normalized := version
	if semverRe.MatchString(normalized) && !strings.HasPrefix(normalized, "v") {
		normalized = "v" + normalized
	}
	return fmt.Sprintf("%s (%s, %s/%s)", normalized, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// sanitizeKey removes non-alphabetic characters and lowercases the string
func sanitizeKey(s string) string {
	s = nonAlpha.ReplaceAllString(s, "")
	return strings.ToLower(s)
}

// loadCSV reads the CSV at path and returns a mapping from sanitized acronym to its entries
func loadCSV(path string) (map[string][]AcronymEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	entries := make(map[string][]AcronymEntry)
	for idx, rec := range recs {
		if len(rec) < 3 {
			continue // skip malformed lines
		}
		// Skip header row if present
		if idx == 0 && strings.EqualFold(rec[0], "acronym") {
			continue
		}

		key := sanitizeKey(rec[0])
		entry := AcronymEntry{
			Full:        strings.TrimSpace(rec[1]),
			Description: strings.TrimSpace(rec[2]),
		}
		entries[key] = append(entries[key], entry)
	}
	return entries, nil
}

func main() {
	// Set the build version from the build info if not set by the build system
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	// CLI flag for Streamable HTTP transport
	var httpAddr string
	flag.StringVar(&httpAddr, "http", "", "run in Streamable HTTP transport on the given address, e.g. :8080")
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("mcp-acronym-lookup version %s\n", buildVersionOutput(Version))
		os.Exit(0)
	}

	// Path to CSV file from environment
	csvPath := os.Getenv("ACRONYM_FILE")
	if csvPath == "" {
		fmt.Fprintln(os.Stderr, "Error: ACRONYM_FILE environment variable is required")
		os.Exit(1)
	}

	// Load acronym entries
	entries, err := loadCSV(csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading CSV data: %v\n", err)
		os.Exit(1)
	}

	// Initialize MCP server with fixed name and version
	srv := mcp.NewServer(&mcp.Implementation{Name: "mcp-acronym-lookup", Version: Version}, nil)

	// Register lookup tool
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "lookupAcronym",
		Description: "Resolve an acronym or initialism to its full form(s) and description(s).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Lookup Acronym",
			ReadOnlyHint:    true,
			DestructiveHint: new(true),
			IdempotentHint:  false,
			OpenWorldHint:   new(true),
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input LookupAcronymInput) (
		*mcp.CallToolResult, LookupAcronymOutput, error,
	) {
		key := sanitizeKey(input.Acronym)
		matches, found := entries[key]
		if !found || len(matches) == 0 {
			return nil, LookupAcronymOutput{}, fmt.Errorf("no entry found for '%s'", input.Acronym)
		}
		return nil, LookupAcronymOutput{Acronym: key, Definitions: matches}, nil
	})

	// Choose transport mode
	if httpAddr != "" {
		fmt.Printf("Starting MCP server using Streamable HTTP transport on %s\n", httpAddr)
		fmt.Printf("Streamable HTTP Endpoint: http://localhost:%s/mcp\n", httpAddr)

		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return srv
		}, &mcp.StreamableHTTPOptions{})

		// Start the server
		if err := http.ListenAndServe(":"+httpAddr, handler); err != nil {
			log.Fatalf("Streamable HTTP server failed to start: %v", err)
		}
	} else {
		// stdio mode by default
		if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatalf("Fatal: MCP server terminated: %v\n", err)
		}
	}
}
