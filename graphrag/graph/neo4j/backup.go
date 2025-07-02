package neo4j

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// BackupData represents the structure of backup data
type BackupData struct {
	Format        string                   `json:"format"`
	GraphName     string                   `json:"graph_name"`
	Metadata      map[string]interface{}   `json:"metadata"`
	Nodes         []map[string]interface{} `json:"nodes"`
	Relationships []map[string]interface{} `json:"relationships"`
}

// Backup creates a backup of the graph
func (s *Store) Backup(ctx context.Context, writer io.Writer, opts *types.GraphBackupOptions) error {
	// Check basic conditions without holding lock
	if !s.connected {
		return fmt.Errorf("not connected to Neo4j server")
	}

	if opts == nil {
		return fmt.Errorf("backup options cannot be nil")
	}

	if opts.GraphName == "" {
		return fmt.Errorf("graph name cannot be empty")
	}

	if writer == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	// Check if graph exists
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("graph %s does not exist", opts.GraphName)
	}

	// Determine format (default to json)
	format := "json"
	if opts.Format != "" {
		format = strings.ToLower(opts.Format)
	}

	// Use critical operation semaphore to serialize data backup and avoid conflicts
	var backupData []byte
	err = executeCriticalOperation(ctx, func() error {
		switch format {
		case "json":
			var backupErr error
			backupData, backupErr = s.backupToJSON(ctx, opts.GraphName, opts)
			return backupErr
		case "cypher":
			var backupErr error
			backupData, backupErr = s.backupToCypher(ctx, opts.GraphName, opts)
			return backupErr
		default:
			return fmt.Errorf("unsupported backup format: %s", format)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Choose writer based on compression option
	var finalWriter io.Writer = writer
	var gzipWriter *gzip.Writer

	if opts.Compress {
		gzipWriter = gzip.NewWriter(writer)
		finalWriter = gzipWriter
		defer func() {
			if gzipWriter != nil {
				gzipWriter.Close()
			}
		}()
	}

	// Write backup data
	if _, err := finalWriter.Write(backupData); err != nil {
		return fmt.Errorf("failed to write backup data: %w", err)
	}

	// Flush gzip writer if used
	if gzipWriter != nil {
		if err := gzipWriter.Close(); err != nil {
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}
		gzipWriter = nil // Prevent double close in defer
	}

	return nil
}

// Restore restores a graph from backup
func (s *Store) Restore(ctx context.Context, reader io.Reader, opts *types.GraphRestoreOptions) error {
	// Check basic conditions without holding lock
	if !s.connected {
		return fmt.Errorf("not connected to Neo4j server")
	}

	if opts == nil {
		return fmt.Errorf("restore options cannot be nil")
	}

	if opts.GraphName == "" {
		return fmt.Errorf("graph name cannot be empty")
	}

	if reader == nil {
		return fmt.Errorf("reader cannot be nil")
	}

	// Check if graph already exists
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}

	if exists && !opts.Force {
		return fmt.Errorf("graph %s already exists, use Force=true to overwrite", opts.GraphName)
	}

	// Try to detect if data is gzip compressed by reading first few bytes
	var finalReader io.Reader

	// Create a buffer to peek at the data
	peekBuffer := make([]byte, 2)
	n, err := io.ReadFull(reader, peekBuffer)
	if err != nil && err != io.ErrUnexpectedEOF {
		// Check if it's EOF (empty data)
		if err == io.EOF {
			return fmt.Errorf("invalid backup data: empty data")
		}
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Check for gzip magic number (0x1f, 0x8b)
	isGzipped := n >= 2 && peekBuffer[0] == 0x1f && peekBuffer[1] == 0x8b

	if isGzipped {
		// Recreate reader with the peeked data
		finalReader = io.MultiReader(
			&singleByteReader{data: peekBuffer[:n]},
			reader,
		)

		gzipReader, err := gzip.NewReader(finalReader)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		finalReader = gzipReader
	} else {
		// Not gzipped, use original reader with peeked data
		finalReader = io.MultiReader(
			&singleByteReader{data: peekBuffer[:n]},
			reader,
		)
	}

	// Read backup data
	backupData, err := io.ReadAll(finalReader)
	if err != nil {
		return fmt.Errorf("failed to read backup data: %w", err)
	}

	// Validate backup data
	if len(backupData) == 0 {
		return fmt.Errorf("invalid backup data: empty data")
	}

	// Create graph if it doesn't exist and createGraph is enabled
	if !exists && opts.CreateGraph {
		if err := s.CreateGraph(ctx, opts.GraphName); err != nil {
			return fmt.Errorf("failed to create graph: %w", err)
		}
		// Wait for database to become available (separate database mode)
		if err := s.waitForDatabaseAvailable(ctx, opts.GraphName, 5*time.Second); err != nil {
			return fmt.Errorf("database not available after creation: %w", err)
		}
	} else if exists && opts.Force {
		// Drop existing graph if force is enabled
		if err := s.DropGraph(ctx, opts.GraphName); err != nil {
			return fmt.Errorf("failed to drop existing graph: %w", err)
		}
		// Recreate the graph
		if err := s.CreateGraph(ctx, opts.GraphName); err != nil {
			return fmt.Errorf("failed to recreate graph: %w", err)
		}
		// Wait for database to become available (separate database mode)
		if err := s.waitForDatabaseAvailable(ctx, opts.GraphName, 5*time.Second); err != nil {
			return fmt.Errorf("database not available after recreation: %w", err)
		}
	}

	// Determine format
	format := "json"
	if opts.Format != "" {
		format = strings.ToLower(opts.Format)
	}

	// Use critical operation semaphore to serialize data restore and avoid conflicts
	return executeCriticalOperation(ctx, func() error {
		// Restore based on format
		switch format {
		case "json":
			return s.restoreFromJSON(ctx, opts.GraphName, backupData)
		case "cypher":
			return s.restoreFromCypher(ctx, opts.GraphName, backupData)
		default:
			return fmt.Errorf("unsupported restore format: %s", format)
		}
	})
}

// backupToJSON creates a JSON backup of the graph
func (s *Store) backupToJSON(ctx context.Context, graphName string, opts *types.GraphBackupOptions) ([]byte, error) {
	var session neo4j.SessionWithContext

	if s.useSeparateDatabase {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: graphName,
		})
	} else {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
	}
	defer session.Close(ctx)

	// Build filter query
	var nodeFilter, relFilter string
	if opts.Filter != nil {
		if nf, ok := opts.Filter["nodes"]; ok {
			if nfStr, ok := nf.(string); ok {
				nodeFilter = nfStr
			}
		}
		if rf, ok := opts.Filter["relationships"]; ok {
			if rfStr, ok := rf.(string); ok {
				relFilter = rfStr
			}
		}
	}

	// Export nodes
	var nodeQuery string
	if s.useSeparateDatabase {
		nodeQuery = "MATCH (n) RETURN n"
		if nodeFilter != "" {
			nodeQuery = fmt.Sprintf("MATCH (n) WHERE %s RETURN n", nodeFilter)
		}
	} else {
		graphLabel := s.GetGraphLabel(graphName)
		nodeQuery = fmt.Sprintf("MATCH (n:%s) RETURN n", graphLabel)
		if nodeFilter != "" {
			nodeQuery = fmt.Sprintf("MATCH (n:%s) WHERE %s RETURN n", graphLabel, nodeFilter)
		}
	}

	nodeResult, err := session.Run(ctx, nodeQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to export nodes: %w", err)
	}

	var nodes []map[string]interface{}
	for nodeResult.Next(ctx) {
		record := nodeResult.Record()
		if node, ok := record.Get("n"); ok {
			if nodeData, ok := node.(neo4j.Node); ok {
				nodes = append(nodes, s.convertNodeToMap(nodeData))
			}
		}
	}

	if err := nodeResult.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate nodes: %w", err)
	}

	// Export relationships
	var relQuery string
	if s.useSeparateDatabase {
		relQuery = "MATCH (a)-[r]->(b) RETURN r, a, b"
		if relFilter != "" {
			relQuery = fmt.Sprintf("MATCH (a)-[r]->(b) WHERE %s RETURN r, a, b", relFilter)
		}
	} else {
		graphLabel := s.GetGraphLabel(graphName)
		relQuery = fmt.Sprintf("MATCH (a:%s)-[r]->(b:%s) RETURN r, a, b", graphLabel, graphLabel)
		if relFilter != "" {
			relQuery = fmt.Sprintf("MATCH (a:%s)-[r]->(b:%s) WHERE %s RETURN r, a, b", graphLabel, graphLabel, relFilter)
		}
	}

	relResult, err := session.Run(ctx, relQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to export relationships: %w", err)
	}

	var relationships []map[string]interface{}
	for relResult.Next(ctx) {
		record := relResult.Record()
		if rel, ok := record.Get("r"); ok {
			if startNode, ok := record.Get("a"); ok {
				if endNode, ok := record.Get("b"); ok {
					if relData, ok := rel.(neo4j.Relationship); ok {
						if startNodeData, ok := startNode.(neo4j.Node); ok {
							if endNodeData, ok := endNode.(neo4j.Node); ok {
								relationships = append(relationships, s.convertRelationshipToMap(relData, startNodeData, endNodeData))
							}
						}
					}
				}
			}
		}
	}

	if err := relResult.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate relationships: %w", err)
	}

	// Create backup data structure
	backupData := BackupData{
		Format:    "json",
		GraphName: graphName,
		Metadata: map[string]interface{}{
			"storage_type":       s.getStorageType(),
			"export_timestamp":   fmt.Sprintf("%d", ctx.Value("timestamp")),
			"node_count":         len(nodes),
			"relationship_count": len(relationships),
		},
		Nodes:         nodes,
		Relationships: relationships,
	}

	// Add extra metadata from options
	if opts.ExtraParams != nil {
		for k, v := range opts.ExtraParams {
			backupData.Metadata[k] = v
		}
	}

	return json.Marshal(backupData)
}

// backupToCypher creates a Cypher script backup of the graph
func (s *Store) backupToCypher(ctx context.Context, graphName string, opts *types.GraphBackupOptions) ([]byte, error) {
	var session neo4j.SessionWithContext

	if s.useSeparateDatabase {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: graphName,
		})
	} else {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
	}
	defer session.Close(ctx)

	var cypherScript strings.Builder

	// Add header comment
	cypherScript.WriteString(fmt.Sprintf("// Neo4j Graph Backup - %s\n", graphName))
	cypherScript.WriteString(fmt.Sprintf("// Storage Type: %s\n", s.getStorageType()))
	cypherScript.WriteString("// Generated by Yao GraphRAG\n\n")

	// Export nodes as CREATE statements
	var nodeQuery string
	if s.useSeparateDatabase {
		nodeQuery = "MATCH (n) RETURN n"
	} else {
		graphLabel := s.GetGraphLabel(graphName)
		nodeQuery = fmt.Sprintf("MATCH (n:%s) RETURN n", graphLabel)
	}

	nodeResult, err := session.Run(ctx, nodeQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to export nodes: %w", err)
	}

	cypherScript.WriteString("// Create Nodes\n")
	for nodeResult.Next(ctx) {
		record := nodeResult.Record()
		if node, ok := record.Get("n"); ok {
			if nodeData, ok := node.(neo4j.Node); ok {
				cypherScript.WriteString(s.convertNodeToCypher(nodeData, graphName))
				cypherScript.WriteString("\n")
			}
		}
	}

	// Export relationships as CREATE statements
	var relQuery string
	if s.useSeparateDatabase {
		relQuery = "MATCH (a)-[r]->(b) RETURN r, a, b"
	} else {
		graphLabel := s.GetGraphLabel(graphName)
		relQuery = fmt.Sprintf("MATCH (a:%s)-[r]->(b:%s) RETURN r, a, b", graphLabel, graphLabel)
	}

	relResult, err := session.Run(ctx, relQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to export relationships: %w", err)
	}

	cypherScript.WriteString("\n// Create Relationships\n")
	for relResult.Next(ctx) {
		record := relResult.Record()
		if rel, ok := record.Get("r"); ok {
			if startNode, ok := record.Get("a"); ok {
				if endNode, ok := record.Get("b"); ok {
					if relData, ok := rel.(neo4j.Relationship); ok {
						if startNodeData, ok := startNode.(neo4j.Node); ok {
							if endNodeData, ok := endNode.(neo4j.Node); ok {
								cypherScript.WriteString(s.convertRelationshipToCypher(relData, startNodeData, endNodeData, graphName))
								cypherScript.WriteString("\n")
							}
						}
					}
				}
			}
		}
	}

	return []byte(cypherScript.String()), nil
}

// restoreFromJSON restores a graph from JSON backup data
func (s *Store) restoreFromJSON(ctx context.Context, graphName string, data []byte) error {
	var backupData BackupData
	if err := json.Unmarshal(data, &backupData); err != nil {
		return fmt.Errorf("failed to parse JSON backup data: %w", err)
	}

	var session neo4j.SessionWithContext

	if s.useSeparateDatabase {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: graphName,
		})
	} else {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
	}
	defer session.Close(ctx)

	// Restore nodes
	for _, nodeData := range backupData.Nodes {
		if err := s.createNodeFromMap(ctx, session, nodeData, graphName); err != nil {
			return fmt.Errorf("failed to restore node: %w", err)
		}
	}

	// Restore relationships
	for _, relData := range backupData.Relationships {
		if err := s.createRelationshipFromMap(ctx, session, relData, graphName); err != nil {
			return fmt.Errorf("failed to restore relationship: %w", err)
		}
	}

	return nil
}

// restoreFromCypher restores a graph from Cypher script backup data
func (s *Store) restoreFromCypher(ctx context.Context, graphName string, data []byte) error {
	var session neo4j.SessionWithContext

	if s.useSeparateDatabase {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: graphName,
		})
	} else {
		session = s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: DefaultDatabase,
		})
	}
	defer session.Close(ctx)

	// Get the original and target graph labels for replacement
	cypherScript := string(data)
	var originalGraphLabel string
	var targetGraphLabel string

	if !s.useSeparateDatabase {
		// Extract original graph name from script header
		lines := strings.Split(cypherScript, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "// Neo4j Graph Backup - ") {
				originalGraphName := strings.TrimPrefix(line, "// Neo4j Graph Backup - ")
				originalGraphLabel = s.getGraphLabelPrefix() + originalGraphName
				targetGraphLabel = s.getGraphLabelPrefix() + graphName
				break
			}
		}

		// Replace graph labels in the script
		if originalGraphLabel != "" && targetGraphLabel != "" && originalGraphLabel != targetGraphLabel {
			cypherScript = strings.ReplaceAll(cypherScript, originalGraphLabel, targetGraphLabel)
		}
	}

	// Split the Cypher script into individual statements
	statements := strings.Split(cypherScript, "\n")

	for _, statement := range statements {
		statement = strings.TrimSpace(statement)

		// Skip empty lines and comments
		if statement == "" || strings.HasPrefix(statement, "//") {
			continue
		}

		// Execute the Cypher statement
		if _, err := session.Run(ctx, statement, nil); err != nil {
			return fmt.Errorf("failed to execute Cypher statement '%s': %w", statement, err)
		}
	}

	return nil
}

// Helper functions

func (s *Store) getStorageType() string {
	if s.useSeparateDatabase {
		return "separate_database"
	}
	return "label_based"
}

func (s *Store) convertNodeToMap(node neo4j.Node) map[string]interface{} {
	result := map[string]interface{}{
		"id":         node.GetId(),
		"element_id": node.GetElementId(),
		"labels":     node.Labels,
		"properties": node.Props,
	}
	return result
}

func (s *Store) convertRelationshipToMap(rel neo4j.Relationship, startNode, endNode neo4j.Node) map[string]interface{} {
	// Get business IDs from node properties
	startBusinessID := ""
	endBusinessID := ""

	if startNode.Props != nil {
		if id, ok := startNode.Props["id"]; ok {
			if idStr, ok := id.(string); ok {
				startBusinessID = idStr
			}
		}
	}

	if endNode.Props != nil {
		if id, ok := endNode.Props["id"]; ok {
			if idStr, ok := id.(string); ok {
				endBusinessID = idStr
			}
		}
	}

	result := map[string]interface{}{
		"id":                rel.GetId(),
		"element_id":        rel.GetElementId(),
		"type":              rel.Type,
		"properties":        rel.Props,
		"start_node":        startNode.GetElementId(),
		"end_node":          endNode.GetElementId(),
		"start_node_id":     startNode.GetId(),
		"end_node_id":       endNode.GetId(),
		"start_business_id": startBusinessID,
		"end_business_id":   endBusinessID,
	}
	return result
}

func (s *Store) convertNodeToCypher(node neo4j.Node, graphName string) string {
	labels := make([]string, len(node.Labels))
	copy(labels, node.Labels)

	if !s.useSeparateDatabase {
		// Add graph label for label-based storage, but only if it's not already present
		graphLabel := s.GetGraphLabel(graphName)
		hasGraphLabel := false
		for _, label := range labels {
			if label == graphLabel {
				hasGraphLabel = true
				break
			}
		}
		if !hasGraphLabel {
			labels = append(labels, graphLabel)
		}
	}

	labelStr := ""
	if len(labels) > 0 {
		labelStr = ":" + strings.Join(labels, ":")
	}

	props := node.Props
	propsStr := ""
	if len(props) > 0 {
		var propParts []string
		for key, value := range props {
			switch v := value.(type) {
			case string:
				propParts = append(propParts, fmt.Sprintf("%s: %q", key, v))
			case int, int64, float64, bool:
				propParts = append(propParts, fmt.Sprintf("%s: %v", key, v))
			default:
				// For complex types, serialize to JSON string
				jsonBytes, _ := json.Marshal(v)
				propParts = append(propParts, fmt.Sprintf("%s: %q", key, string(jsonBytes)))
			}
		}
		if len(propParts) > 0 {
			propsStr = " {" + strings.Join(propParts, ", ") + "}"
		}
	}

	return fmt.Sprintf("CREATE (n%s%s);", labelStr, propsStr)
}

func (s *Store) convertRelationshipToCypher(rel neo4j.Relationship, startNode, endNode neo4j.Node, graphName string) string {
	// For relationships, we need to match the nodes first
	startLabels := make([]string, len(startNode.Labels))
	copy(startLabels, startNode.Labels)
	endLabels := make([]string, len(endNode.Labels))
	copy(endLabels, endNode.Labels)

	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(graphName)

		// Check if start node already has graph label
		hasStartGraphLabel := false
		for _, label := range startLabels {
			if label == graphLabel {
				hasStartGraphLabel = true
				break
			}
		}
		if !hasStartGraphLabel {
			startLabels = append(startLabels, graphLabel)
		}

		// Check if end node already has graph label
		hasEndGraphLabel := false
		for _, label := range endLabels {
			if label == graphLabel {
				hasEndGraphLabel = true
				break
			}
		}
		if !hasEndGraphLabel {
			endLabels = append(endLabels, graphLabel)
		}
	}

	startLabelStr := ""
	if len(startLabels) > 0 {
		startLabelStr = ":" + strings.Join(startLabels, ":")
	}

	endLabelStr := ""
	if len(endLabels) > 0 {
		endLabelStr = ":" + strings.Join(endLabels, ":")
	}

	props := rel.Props
	propsStr := ""
	if len(props) > 0 {
		var propParts []string
		for key, value := range props {
			switch v := value.(type) {
			case string:
				propParts = append(propParts, fmt.Sprintf("%s: %q", key, v))
			case int, int64, float64, bool:
				propParts = append(propParts, fmt.Sprintf("%s: %v", key, v))
			default:
				// For complex types, serialize to JSON string
				jsonBytes, _ := json.Marshal(v)
				propParts = append(propParts, fmt.Sprintf("%s: %q", key, string(jsonBytes)))
			}
		}
		if len(propParts) > 0 {
			propsStr = " {" + strings.Join(propParts, ", ") + "}"
		}
	}

	// Use business IDs from node properties for better portability
	startBusinessID := ""
	endBusinessID := ""

	if startNode.Props != nil {
		if id, ok := startNode.Props["id"]; ok {
			if idStr, ok := id.(string); ok {
				startBusinessID = idStr
			}
		}
	}

	if endNode.Props != nil {
		if id, ok := endNode.Props["id"]; ok {
			if idStr, ok := id.(string); ok {
				endBusinessID = idStr
			}
		}
	}

	if startBusinessID != "" && endBusinessID != "" {
		return fmt.Sprintf("MATCH (a%s), (b%s) WHERE a.id = '%s' AND b.id = '%s' CREATE (a)-[:%s%s]->(b);",
			startLabelStr, endLabelStr, startBusinessID, endBusinessID, rel.Type, propsStr)
	}
	// Fallback to elementId if business ID is not available
	return fmt.Sprintf("MATCH (a%s), (b%s) WHERE elementId(a) = '%s' AND elementId(b) = '%s' CREATE (a)-[:%s%s]->(b);",
		startLabelStr, endLabelStr, startNode.GetElementId(), endNode.GetElementId(), rel.Type, propsStr)
}

func (s *Store) createNodeFromMap(ctx context.Context, session neo4j.SessionWithContext, nodeData map[string]interface{}, graphName string) error {
	labels, _ := nodeData["labels"].([]interface{})
	properties, _ := nodeData["properties"].(map[string]interface{})

	var labelStrs []string
	for _, label := range labels {
		if labelStr, ok := label.(string); ok {
			labelStrs = append(labelStrs, labelStr)
		}
	}

	if !s.useSeparateDatabase {
		// Add graph label for label-based storage
		graphLabel := s.GetGraphLabel(graphName)
		labelStrs = append(labelStrs, graphLabel)
	}

	labelStr := ""
	if len(labelStrs) > 0 {
		labelStr = ":" + strings.Join(labelStrs, ":")
	}

	query := fmt.Sprintf("CREATE (n%s) SET n = $props", labelStr)
	_, err := session.Run(ctx, query, map[string]interface{}{
		"props": properties,
	})

	return err
}

func (s *Store) createRelationshipFromMap(ctx context.Context, session neo4j.SessionWithContext, relData map[string]interface{}, graphName string) error {
	relType, _ := relData["type"].(string)
	properties, _ := relData["properties"].(map[string]interface{})

	// Try to get business IDs first (preferred method)
	startBusinessID, hasStartBusinessID := relData["start_business_id"].(string)
	endBusinessID, hasEndBusinessID := relData["end_business_id"].(string)

	// Fallback to element IDs
	startElementID, _ := relData["start_node"].(string)
	endElementID, _ := relData["end_node"].(string)

	var query string
	var params map[string]interface{}

	if s.useSeparateDatabase {
		if hasStartBusinessID && hasEndBusinessID && startBusinessID != "" && endBusinessID != "" {
			// Use business ID matching (preferred)
			query = fmt.Sprintf("MATCH (a), (b) WHERE a.id = $startId AND b.id = $endId CREATE (a)-[r:%s]->(b) SET r = $props", relType)
			params = map[string]interface{}{
				"startId": startBusinessID,
				"endId":   endBusinessID,
				"props":   properties,
			}
		} else {
			// Fallback to element ID
			query = fmt.Sprintf("MATCH (a), (b) WHERE elementId(a) = $startId AND elementId(b) = $endId CREATE (a)-[r:%s]->(b) SET r = $props", relType)
			params = map[string]interface{}{
				"startId": startElementID,
				"endId":   endElementID,
				"props":   properties,
			}
		}
	} else {
		graphLabel := s.GetGraphLabel(graphName)
		if hasStartBusinessID && hasEndBusinessID && startBusinessID != "" && endBusinessID != "" {
			// Use business ID matching (preferred)
			query = fmt.Sprintf("MATCH (a:%s), (b:%s) WHERE a.id = $startId AND b.id = $endId CREATE (a)-[r:%s]->(b) SET r = $props", graphLabel, graphLabel, relType)
			params = map[string]interface{}{
				"startId": startBusinessID,
				"endId":   endBusinessID,
				"props":   properties,
			}
		} else {
			// Fallback to element ID
			query = fmt.Sprintf("MATCH (a:%s), (b:%s) WHERE elementId(a) = $startId AND elementId(b) = $endId CREATE (a)-[r:%s]->(b) SET r = $props", graphLabel, graphLabel, relType)
			params = map[string]interface{}{
				"startId": startElementID,
				"endId":   endElementID,
				"props":   properties,
			}
		}
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// waitForDatabaseAvailable waits for a database to become available in separate database mode
func (s *Store) waitForDatabaseAvailable(ctx context.Context, databaseName string, timeout time.Duration) error {
	if !s.useSeparateDatabase {
		return nil // No need to wait in label-based mode
	}

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		// Try to create a session with the database
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{
			DatabaseName: databaseName,
		})

		// Try a simple query
		_, err := session.Run(ctx, "RETURN 1", nil)
		session.Close(ctx)

		if err == nil {
			return nil // Database is available
		}

		lastErr = err

		// Check if it's a database unavailable error or connection related
		if strings.Contains(err.Error(), "DatabaseUnavailable") ||
			strings.Contains(err.Error(), "routing table") ||
			strings.Contains(err.Error(), "database is unavailable") {
			time.Sleep(100 * time.Millisecond) // Increased sleep time
			continue
		}

		// For other errors, still retry but with shorter intervals
		time.Sleep(50 * time.Millisecond)
	}

	if lastErr != nil {
		return fmt.Errorf("database %s did not become available within %v, last error: %w", databaseName, timeout, lastErr)
	}
	return fmt.Errorf("database %s did not become available within %v", databaseName, timeout)
}

// singleByteReader is a helper to read peeked bytes
type singleByteReader struct {
	data []byte
	pos  int
}

func (r *singleByteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n

	if r.pos >= len(r.data) {
		err = io.EOF
	}

	return n, err
}
