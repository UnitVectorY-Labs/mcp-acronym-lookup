package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AcronymEntry represents one row in the CSV
type AcronymEntry struct {
	Full        string `json:"full"`
	Description string `json:"description"`
}

var nonAlpha = regexp.MustCompile("[^A-Za-z]+")

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
	// CLI flag for SSE/HTTP mode
	var sseAddr string
	flag.StringVar(&sseAddr, "sse", "", "run in SSE (HTTP/SSE) mode on the given address, e.g. :8080")
	flag.Parse()

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
	srv := server.NewMCPServer("mcp-acronym-lookup", "0.1.0")

	// Register lookup tool
	tool := mcp.NewTool(
		"lookupAcronym",
		mcp.WithDescription("Resolve an acronym or initialism to its full form(s) and description(s)."),
		mcp.WithString("acronym", mcp.Description("The acronym or initialism to resolve."), mcp.Required()),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		acronym, ok := args["acronym"].(string)
		if !ok {
			return mcp.NewToolResultError("invalid or missing 'acronym' parameter"), nil
		}
		key := sanitizeKey(acronym)
		matches, found := entries[key]
		if !found || len(matches) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("no entry found for '%s'", acronym)), nil
		}
		// Prepare response
		resp := map[string]interface{}{
			"acronym":     key,
			"definitions": matches,
		}
		data, err := json.Marshal(resp)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to encode response", err), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	// Choose mode
	if sseAddr != "" {
		fmt.Printf("Starting MCP server in SSE mode on %s\n", sseAddr)
		sseSrv := server.NewSSEServer(
			srv,
			server.WithStaticBasePath("/"),
			server.WithSSEEndpoint("/mcp/sse"),
			server.WithMessageEndpoint("/mcp/message"),
		)
		mux := http.NewServeMux()
		mux.Handle("/", sseSrv)

		fmt.Printf("SSE Endpoint: %s\n", sseSrv.CompleteSsePath())
		fmt.Printf("Message Endpoint: %s\n", sseSrv.CompleteMessagePath())

		httpSrv := &http.Server{
			Addr:    sseAddr,
			Handler: mux,
		}
		log.Fatal(httpSrv.ListenAndServe())
	} else {
		// stdio mode
		if err := server.ServeStdio(srv); err != nil {
			log.Fatalf("Fatal: MCP server terminated: %v\n", err)
		}
	}
}
