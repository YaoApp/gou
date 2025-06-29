package neo4j

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yaoapp/gou/graphrag/types"
)

// Query executes a graph query with flexible options
func (s *Store) Query(ctx context.Context, opts *types.GraphQueryOptions) (*types.GraphResult, error) {
	s.mu.RLock()
	connected := s.connected
	s.mu.RUnlock()

	if !connected {
		return nil, fmt.Errorf("store is not connected")
	}

	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	if opts.GraphName == "" {
		return nil, fmt.Errorf("graph name cannot be empty")
	}

	// Validate graph name
	if !isValidGraphName(opts.GraphName) {
		return nil, fmt.Errorf("invalid graph name: %s (only alphanumeric, underscore, and dash allowed)", opts.GraphName)
	}

	// Set timeout context if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.Timeout)*time.Second)
		defer cancel()
	}

	// Ensure the graph exists
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("graph '%s' does not exist", opts.GraphName)
	}

	// Route to specific query implementation based on query type
	switch strings.ToLower(opts.QueryType) {
	case "cypher":
		return s.executeCypherQuery(ctx, opts)
	case "traversal":
		return s.executeTraversalQuery(ctx, opts)
	case "path":
		return s.executePathQuery(ctx, opts)
	case "analytics":
		return s.executeAnalyticsQuery(ctx, opts)
	case "custom", "":
		// Default to custom/direct query execution
		return s.executeCustomQuery(ctx, opts)
	default:
		return nil, fmt.Errorf("unsupported query type: %s", opts.QueryType)
	}
}

// Communities performs community detection and analysis
func (s *Store) Communities(ctx context.Context, opts *types.CommunityDetectionOptions) ([]*types.Community, error) {
	s.mu.RLock()
	connected := s.connected
	s.mu.RUnlock()

	if !connected {
		return nil, fmt.Errorf("store is not connected")
	}

	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	if opts.GraphName == "" {
		return nil, fmt.Errorf("graph name cannot be empty")
	}

	// Validate graph name
	if !isValidGraphName(opts.GraphName) {
		return nil, fmt.Errorf("invalid graph name: %s (only alphanumeric, underscore, and dash allowed)", opts.GraphName)
	}

	// Ensure the graph exists
	exists, err := s.GraphExists(ctx, opts.GraphName)
	if err != nil {
		return nil, fmt.Errorf("failed to check graph existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("graph '%s' does not exist", opts.GraphName)
	}

	// Route to specific community detection algorithm
	switch strings.ToLower(opts.Algorithm) {
	case "leiden":
		return s.executeLeidenCommunityDetection(ctx, opts)
	case "louvain":
		return s.executeLouvainCommunityDetection(ctx, opts)
	case "label_propagation":
		return s.executeLabelPropagationCommunityDetection(ctx, opts)
	default:
		return nil, fmt.Errorf("unsupported community detection algorithm: %s", opts.Algorithm)
	}
}

// executeCypherQuery executes a Cypher query
func (s *Store) executeCypherQuery(ctx context.Context, opts *types.GraphQueryOptions) (*types.GraphResult, error) {
	if opts.Query == "" {
		return nil, fmt.Errorf("cypher query cannot be empty")
	}

	// Choose session configuration based on separate database mode
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// Prepare query parameters
	parameters := opts.Parameters
	if parameters == nil {
		parameters = make(map[string]interface{})
	}

	// For label-based mode, add graph label parameter if needed
	if !s.useSeparateDatabase {
		parameters["__graph_label"] = s.GetGraphLabel(opts.GraphName)
		parameters["__graph_namespace"] = opts.GraphName
	}

	// Execute query with appropriate access mode and ensure result is consumed within transaction
	var graphResult *types.GraphResult

	// Determine if this is a read-only query (default to read-only if not specified)
	isReadOnly := opts.ReadOnly
	if !opts.ReadOnly {
		// Auto-detect read-only queries by checking for common read-only patterns
		queryUpper := strings.ToUpper(strings.TrimSpace(opts.Query))
		if strings.HasPrefix(queryUpper, "MATCH") ||
			strings.HasPrefix(queryUpper, "RETURN") ||
			strings.HasPrefix(queryUpper, "WITH") ||
			strings.HasPrefix(queryUpper, "UNWIND") ||
			strings.HasPrefix(queryUpper, "CALL") ||
			(strings.Contains(queryUpper, "MATCH") && !strings.Contains(queryUpper, "CREATE") &&
				!strings.Contains(queryUpper, "MERGE") && !strings.Contains(queryUpper, "DELETE") &&
				!strings.Contains(queryUpper, "SET") && !strings.Contains(queryUpper, "REMOVE")) {
			isReadOnly = true
		}
	}

	if isReadOnly {
		// Use read transaction
		result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			result, err := tx.Run(ctx, opts.Query, parameters)
			if err != nil {
				return nil, err
			}
			return s.parseQueryResult(result, opts.ReturnType)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to execute read query: %w", err)
		}
		graphResult = result.(*types.GraphResult)
	} else {
		// Use write transaction
		result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			result, err := tx.Run(ctx, opts.Query, parameters)
			if err != nil {
				return nil, err
			}
			return s.parseQueryResult(result, opts.ReturnType)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to execute write query: %w", err)
		}
		graphResult = result.(*types.GraphResult)
	}

	return graphResult, nil
}

// executeTraversalQuery executes a graph traversal query
func (s *Store) executeTraversalQuery(ctx context.Context, opts *types.GraphQueryOptions) (*types.GraphResult, error) {
	// Auto-create TraversalOptions from Parameters if not provided
	if opts.TraversalOptions == nil {
		opts.TraversalOptions = &types.GraphTraversalOptions{}

		// Extract common traversal parameters from Parameters map
		if params := opts.Parameters; params != nil {
			if maxDepth, ok := params["max_depth"].(int); ok {
				opts.TraversalOptions.MaxDepth = maxDepth
			}
			if minDepth, ok := params["min_depth"].(int); ok {
				opts.TraversalOptions.MinDepth = minDepth
			}
			if direction, ok := params["direction"].(string); ok {
				opts.TraversalOptions.Direction = direction
			}
			if returnPaths, ok := params["return_paths"].(bool); ok {
				opts.TraversalOptions.ReturnPaths = returnPaths
			}
			if uniquePaths, ok := params["unique_paths"].(bool); ok {
				opts.TraversalOptions.UniquePaths = uniquePaths
			}
			if limit, ok := params["limit"].(int); ok {
				opts.TraversalOptions.Limit = limit
			}
			// Handle node and relationship filters
			if nodeFilter, ok := params["node_filter"].(map[string]interface{}); ok {
				opts.TraversalOptions.NodeFilter = nodeFilter
			}
			if relFilter, ok := params["rel_filter"].(map[string]interface{}); ok {
				opts.TraversalOptions.RelFilter = relFilter
			}
		}
	}

	// Build traversal query based on options
	query, parameters := s.buildTraversalQuery(opts)

	// Create a new GraphQueryOptions for cypher execution
	cypherOpts := &types.GraphQueryOptions{
		GraphName:  opts.GraphName,
		QueryType:  "cypher",
		Query:      query,
		Parameters: parameters,
		ReadOnly:   opts.ReadOnly,
		ReturnType: opts.ReturnType,
		Limit:      opts.Limit,
		Skip:       opts.Skip,
	}

	return s.executeCypherQuery(ctx, cypherOpts)
}

// executePathQuery executes a path finding query
func (s *Store) executePathQuery(ctx context.Context, opts *types.GraphQueryOptions) (*types.GraphResult, error) {
	// For path queries, we typically need start and end nodes
	startNode, hasStart := opts.Parameters["start_node"]
	endNode, hasEnd := opts.Parameters["end_node"]

	if !hasStart || !hasEnd {
		return nil, fmt.Errorf("path query requires 'start_node' and 'end_node' parameters")
	}

	// Build path query
	query, parameters := s.buildPathQuery(opts, startNode, endNode)

	// Create a new GraphQueryOptions for cypher execution
	cypherOpts := &types.GraphQueryOptions{
		GraphName:  opts.GraphName,
		QueryType:  "cypher",
		Query:      query,
		Parameters: parameters,
		ReadOnly:   true,    // Path queries are always read-only
		ReturnType: "paths", // Force return type to paths
		Limit:      opts.Limit,
		Skip:       opts.Skip,
	}

	return s.executeCypherQuery(ctx, cypherOpts)
}

// executeAnalyticsQuery executes a graph analytics query
func (s *Store) executeAnalyticsQuery(ctx context.Context, opts *types.GraphQueryOptions) (*types.GraphResult, error) {
	// Auto-create AnalyticsOptions from Parameters if not provided
	if opts.AnalyticsOptions == nil {
		opts.AnalyticsOptions = &types.GraphAnalyticsOptions{}

		// Extract common analytics parameters from Parameters map
		if params := opts.Parameters; params != nil {
			if algorithm, ok := params["algorithm"].(string); ok {
				opts.AnalyticsOptions.Algorithm = algorithm
			}
			if iterations, ok := params["iterations"].(int); ok {
				opts.AnalyticsOptions.Iterations = iterations
			}
			if maxIterations, ok := params["max_iterations"].(int); ok {
				opts.AnalyticsOptions.Iterations = maxIterations // max_iterations maps to iterations
			}
			if dampingFactor, ok := params["damping_factor"].(float64); ok {
				opts.AnalyticsOptions.DampingFactor = dampingFactor
			}
			// Copy all parameters to AnalyticsOptions.Parameters
			opts.AnalyticsOptions.Parameters = make(map[string]interface{})
			for k, v := range params {
				opts.AnalyticsOptions.Parameters[k] = v
			}
		}
	}

	// Build analytics query based on algorithm
	query, parameters := s.buildAnalyticsQuery(opts)

	// Create a new GraphQueryOptions for cypher execution
	cypherOpts := &types.GraphQueryOptions{
		GraphName:  opts.GraphName,
		QueryType:  "cypher",
		Query:      query,
		Parameters: parameters,
		ReadOnly:   true, // Analytics queries are always read-only
		ReturnType: opts.ReturnType,
		Limit:      opts.Limit,
		Skip:       opts.Skip,
	}

	return s.executeCypherQuery(ctx, cypherOpts)
}

// executeCustomQuery executes a custom query using the Query field directly
func (s *Store) executeCustomQuery(ctx context.Context, opts *types.GraphQueryOptions) (*types.GraphResult, error) {
	if opts.Query == "" {
		return nil, fmt.Errorf("custom query cannot be empty")
	}

	// For custom queries, treat as Cypher
	cypherOpts := &types.GraphQueryOptions{
		GraphName:  opts.GraphName,
		QueryType:  "cypher",
		Query:      opts.Query,
		Parameters: opts.Parameters,
		ReadOnly:   opts.ReadOnly,
		ReturnType: opts.ReturnType,
		Limit:      opts.Limit,
		Skip:       opts.Skip,
	}

	return s.executeCypherQuery(ctx, cypherOpts)
}

// executeLeidenCommunityDetection executes Leiden community detection algorithm
func (s *Store) executeLeidenCommunityDetection(ctx context.Context, opts *types.CommunityDetectionOptions) ([]*types.Community, error) {
	// Choose session configuration based on separate database mode
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// Build Leiden algorithm query
	query, parameters := s.buildLeidenQuery(opts)

	// Execute the community detection query
	result, err := session.Run(ctx, query, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Leiden community detection: %w", err)
	}

	return s.parseCommunityResult(result, opts)
}

// executeLouvainCommunityDetection executes Louvain community detection algorithm
func (s *Store) executeLouvainCommunityDetection(ctx context.Context, opts *types.CommunityDetectionOptions) ([]*types.Community, error) {
	// Choose session configuration based on separate database mode
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// Build Louvain algorithm query
	query, parameters := s.buildLouvainQuery(opts)

	// Execute the community detection query
	result, err := session.Run(ctx, query, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Louvain community detection: %w", err)
	}

	return s.parseCommunityResult(result, opts)
}

// executeLabelPropagationCommunityDetection executes Label Propagation community detection algorithm
func (s *Store) executeLabelPropagationCommunityDetection(ctx context.Context, opts *types.CommunityDetectionOptions) ([]*types.Community, error) {
	// Choose session configuration based on separate database mode
	sessionConfig := neo4j.SessionConfig{}
	if s.useSeparateDatabase {
		sessionConfig.DatabaseName = opts.GraphName
	} else {
		sessionConfig.DatabaseName = DefaultDatabase
	}

	session := s.driver.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// Build Label Propagation algorithm query
	query, parameters := s.buildLabelPropagationQuery(opts)

	// Execute the community detection query
	result, err := session.Run(ctx, query, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Label Propagation community detection: %w", err)
	}

	return s.parseCommunityResult(result, opts)
}

// ===== Helper Methods =====

// parseQueryResult parses Neo4j query result into GraphResult
func (s *Store) parseQueryResult(result neo4j.ResultWithContext, returnType string) (*types.GraphResult, error) {
	ctx := context.Background()

	graphResult := &types.GraphResult{
		Nodes:         []types.Node{},
		Relationships: []types.Relationship{},
		Paths:         []types.Path{},
		Records:       []interface{}{},
	}

	// Process all records
	for result.Next(ctx) {
		record := result.Record()

		// Add raw record to results
		recordData := make(map[string]interface{})
		for _, key := range record.Keys {
			recordData[key] = record.AsMap()[key]
		}
		graphResult.Records = append(graphResult.Records, recordData)

		// Parse specific return types
		switch strings.ToLower(returnType) {
		case "nodes":
			s.extractNodesFromRecord(record, graphResult)
		case "relationships":
			s.extractRelationshipsFromRecord(record, graphResult)
		case "paths":
			s.extractPathsFromRecord(record, graphResult)
		case "all", "":
			// Try to extract all types
			s.extractNodesFromRecord(record, graphResult)
			s.extractRelationshipsFromRecord(record, graphResult)
			s.extractPathsFromRecord(record, graphResult)
		}
	}

	// Check for query execution errors
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("query execution error: %w", err)
	}

	return graphResult, nil
}

// extractNodesFromRecord extracts nodes from a Neo4j record
func (s *Store) extractNodesFromRecord(record *neo4j.Record, graphResult *types.GraphResult) {
	for _, value := range record.Values {
		if node, ok := value.(neo4j.Node); ok {
			graphNode := s.convertNeo4jNodeToGraphNode(node)
			graphResult.Nodes = append(graphResult.Nodes, graphNode)
		}
	}
}

// extractRelationshipsFromRecord extracts relationships from a Neo4j record
func (s *Store) extractRelationshipsFromRecord(record *neo4j.Record, graphResult *types.GraphResult) {
	for _, value := range record.Values {
		if rel, ok := value.(neo4j.Relationship); ok {
			graphRel := s.convertNeo4jRelationshipToGraphRelationship(rel)
			graphResult.Relationships = append(graphResult.Relationships, graphRel)
		}
	}
}

// extractPathsFromRecord extracts paths from a Neo4j record
func (s *Store) extractPathsFromRecord(record *neo4j.Record, graphResult *types.GraphResult) {
	for _, value := range record.Values {
		if path, ok := value.(neo4j.Path); ok {
			graphPath := s.convertNeo4jPathToGraphPath(path)
			graphResult.Paths = append(graphResult.Paths, graphPath)
		}
	}
}

// convertNeo4jNodeToGraphNode converts Neo4j Node to types.Node
func (s *Store) convertNeo4jNodeToGraphNode(node neo4j.Node) types.Node {
	// Extract ID - handle different ID representations
	var nodeID string
	if id, exists := node.Props["id"]; exists {
		nodeID = fmt.Sprintf("%v", id)
	} else {
		nodeID = node.ElementId
	}

	return types.Node{
		ID:         nodeID,
		Name:       fmt.Sprintf("%v", node.Props["name"]),
		Type:       fmt.Sprintf("%v", node.Props["type"]),
		Labels:     node.Labels,
		Properties: node.Props,
	}
}

// convertNeo4jRelationshipToGraphRelationship converts Neo4j Relationship to types.Relationship
func (s *Store) convertNeo4jRelationshipToGraphRelationship(rel neo4j.Relationship) types.Relationship {
	// Extract ID - handle different ID representations
	var relID string
	if id, exists := rel.Props["id"]; exists {
		relID = fmt.Sprintf("%v", id)
	} else {
		relID = rel.ElementId
	}

	return types.Relationship{
		ID:         relID,
		Type:       rel.Type,
		StartNode:  rel.StartElementId,
		EndNode:    rel.EndElementId,
		Properties: rel.Props,
	}
}

// convertNeo4jPathToGraphPath converts Neo4j Path to types.Path
func (s *Store) convertNeo4jPathToGraphPath(path neo4j.Path) types.Path {
	graphPath := types.Path{
		Nodes:         make([]types.Node, len(path.Nodes)),
		Relationships: make([]types.Relationship, len(path.Relationships)),
		Length:        len(path.Relationships),
	}

	// Convert nodes
	for i, node := range path.Nodes {
		graphPath.Nodes[i] = s.convertNeo4jNodeToGraphNode(node)
	}

	// Convert relationships
	for i, rel := range path.Relationships {
		graphPath.Relationships[i] = s.convertNeo4jRelationshipToGraphRelationship(rel)
	}

	return graphPath
}

// buildTraversalQuery builds a Cypher query for graph traversal
func (s *Store) buildTraversalQuery(opts *types.GraphQueryOptions) (string, map[string]interface{}) {
	traversalOpts := opts.TraversalOptions
	parameters := make(map[string]interface{})

	// Copy original parameters to preserve start_node, relationships, etc.
	for k, v := range opts.Parameters {
		parameters[k] = v
	}

	// Base query parts
	var queryParts []string
	var whereConditions []string

	// Build MATCH clause
	var matchClause string
	if traversalOpts.ReturnPaths {
		// For paths, use variable length patterns
		direction := ""
		switch strings.ToUpper(traversalOpts.Direction) {
		case "INCOMING":
			direction = "<-"
		case "OUTGOING":
			direction = "->"
		case "BOTH", "":
			direction = "-"
		}

		depthPattern := ""
		if traversalOpts.MinDepth > 0 || traversalOpts.MaxDepth > 0 {
			minDepth := traversalOpts.MinDepth
			maxDepth := traversalOpts.MaxDepth
			if maxDepth == 0 {
				maxDepth = 10 // Default max depth
			}
			depthPattern = fmt.Sprintf("*%d..%d", minDepth, maxDepth)
		}

		// Handle relationship type filtering
		relTypeFilter := ""
		if relationships, ok := opts.Parameters["relationships"].([]string); ok && len(relationships) > 0 {
			relTypeFilter = ":" + strings.Join(relationships, "|")
		}

		matchClause = fmt.Sprintf("MATCH p = (start)%s[r%s%s]%s(end)", direction[:1], relTypeFilter, depthPattern, direction[1:])
	} else {
		// Handle relationship type filtering for simple traversal
		relTypeFilter := ""
		if relationships, ok := opts.Parameters["relationships"].([]string); ok && len(relationships) > 0 {
			relTypeFilter = ":" + strings.Join(relationships, "|")
		}

		direction := ""
		switch strings.ToUpper(traversalOpts.Direction) {
		case "INCOMING":
			direction = "<-[r" + relTypeFilter + "]-"
		case "OUTGOING":
			direction = "-[r" + relTypeFilter + "]->"
		case "BOTH", "":
			direction = "-[r" + relTypeFilter + "]-"
		}

		matchClause = fmt.Sprintf("MATCH (start)%s(end)", direction)
	}

	queryParts = append(queryParts, matchClause)

	// Add start node identification
	if startNode, ok := opts.Parameters["start_node"]; ok {
		whereConditions = append(whereConditions, "start.id = $start_node")
		parameters["start_node"] = startNode
	}

	// Add graph filtering for label-based mode
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		whereConditions = append(whereConditions, fmt.Sprintf("start:%s", escapedGraphLabel))
		whereConditions = append(whereConditions, fmt.Sprintf("end:%s", escapedGraphLabel))
	}

	// Add node filtering
	if len(traversalOpts.NodeFilter) > 0 {
		for key, value := range traversalOpts.NodeFilter {
			paramKey := fmt.Sprintf("node_%s", key)
			whereConditions = append(whereConditions, fmt.Sprintf("start.%s = $%s OR end.%s = $%s", key, paramKey, key, paramKey))
			parameters[paramKey] = value
		}
	}

	// Add relationship filtering
	if len(traversalOpts.RelFilter) > 0 {
		for key, value := range traversalOpts.RelFilter {
			paramKey := fmt.Sprintf("rel_%s", key)
			whereConditions = append(whereConditions, fmt.Sprintf("r.%s = $%s", key, paramKey))
			parameters[paramKey] = value
		}
	}

	// Add WHERE clause if needed
	if len(whereConditions) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))
	}

	// Add RETURN clause
	if traversalOpts.ReturnPaths {
		queryParts = append(queryParts, "RETURN p")
	} else {
		queryParts = append(queryParts, "RETURN start, r, end")
	}

	// Add LIMIT if specified
	if traversalOpts.Limit > 0 {
		queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", traversalOpts.Limit))
	}

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildPathQuery builds a Cypher query for path finding
func (s *Store) buildPathQuery(opts *types.GraphQueryOptions, startNode, endNode interface{}) (string, map[string]interface{}) {
	parameters := make(map[string]interface{})
	parameters["start_node"] = startNode
	parameters["end_node"] = endNode

	var queryParts []string
	var whereConditions []string

	// Build MATCH clause for shortest path
	matchClause := "MATCH p = shortestPath((start)-[*]-(end))"
	queryParts = append(queryParts, matchClause)

	// Add node identification
	whereConditions = append(whereConditions, "start.id = $start_node")
	whereConditions = append(whereConditions, "end.id = $end_node")

	// Add graph filtering for label-based mode
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		escapedGraphLabel := "`" + strings.ReplaceAll(graphLabel, "`", "``") + "`"
		whereConditions = append(whereConditions, fmt.Sprintf("start:%s", escapedGraphLabel))
		whereConditions = append(whereConditions, fmt.Sprintf("end:%s", escapedGraphLabel))
	}

	// Add WHERE clause
	queryParts = append(queryParts, "WHERE "+strings.Join(whereConditions, " AND "))

	// Add RETURN clause
	queryParts = append(queryParts, "RETURN p")

	// Add LIMIT if specified
	if opts.Limit > 0 {
		queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", opts.Limit))
	}

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildAnalyticsQuery builds a Cypher query for graph analytics
func (s *Store) buildAnalyticsQuery(opts *types.GraphQueryOptions) (string, map[string]interface{}) {
	analyticsOpts := opts.AnalyticsOptions
	parameters := make(map[string]interface{})

	// Handle different analytics algorithms
	switch strings.ToLower(analyticsOpts.Algorithm) {
	case "pagerank":
		return s.buildPageRankQuery(opts.GraphName, analyticsOpts, parameters)
	case "betweenness":
		return s.buildBetweennessCentralityQuery(opts.GraphName, analyticsOpts, parameters)
	case "closeness":
		return s.buildClosenessCentralityQuery(opts.GraphName, analyticsOpts, parameters)
	case "degree":
		return s.buildDegreeCentralityQuery(opts.GraphName, analyticsOpts, parameters)
	default:
		// Fallback to basic node statistics
		return s.buildBasicStatsQuery(opts.GraphName, parameters)
	}
}

// buildPageRankQuery builds PageRank algorithm query
func (s *Store) buildPageRankQuery(graphName string, opts *types.GraphAnalyticsOptions, parameters map[string]interface{}) (string, map[string]interface{}) {
	var queryParts []string

	// Basic PageRank calculation using APOC if available, otherwise approximation
	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(graphName)
		queryParts = append(queryParts, fmt.Sprintf("MATCH (n:%s)", graphLabel))
	} else {
		queryParts = append(queryParts, "MATCH (n)")
	}

	// Simple degree-based approximation of PageRank
	queryParts = append(queryParts, "OPTIONAL MATCH (n)-[r]-()")
	queryParts = append(queryParts, "WITH n, count(r) as degree")
	queryParts = append(queryParts, "RETURN n.id as node_id, degree, degree * 1.0 / (degree + 1) as pagerank")
	queryParts = append(queryParts, "ORDER BY pagerank DESC")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildBetweennessCentralityQuery builds betweenness centrality query
func (s *Store) buildBetweennessCentralityQuery(graphName string, opts *types.GraphAnalyticsOptions, parameters map[string]interface{}) (string, map[string]interface{}) {
	// Simplified betweenness calculation - in practice would use APOC procedures
	var queryParts []string

	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(graphName)
		queryParts = append(queryParts, fmt.Sprintf("MATCH (n:%s)", graphLabel))
	} else {
		queryParts = append(queryParts, "MATCH (n)")
	}

	queryParts = append(queryParts, "RETURN n.id as node_id, 0.0 as betweenness_centrality")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildClosenessCentralityQuery builds closeness centrality query
func (s *Store) buildClosenessCentralityQuery(graphName string, opts *types.GraphAnalyticsOptions, parameters map[string]interface{}) (string, map[string]interface{}) {
	// Simplified closeness calculation
	var queryParts []string

	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(graphName)
		queryParts = append(queryParts, fmt.Sprintf("MATCH (n:%s)", graphLabel))
	} else {
		queryParts = append(queryParts, "MATCH (n)")
	}

	queryParts = append(queryParts, "RETURN n.id as node_id, 0.0 as closeness_centrality")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildDegreeCentralityQuery builds degree centrality query
func (s *Store) buildDegreeCentralityQuery(graphName string, opts *types.GraphAnalyticsOptions, parameters map[string]interface{}) (string, map[string]interface{}) {
	var queryParts []string

	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(graphName)
		queryParts = append(queryParts, fmt.Sprintf("MATCH (n:%s)", graphLabel))
	} else {
		queryParts = append(queryParts, "MATCH (n)")
	}

	queryParts = append(queryParts, "OPTIONAL MATCH (n)-[r]-()")
	queryParts = append(queryParts, "WITH n, count(r) as degree")
	queryParts = append(queryParts, "RETURN n.id as node_id, degree as degree_centrality")
	queryParts = append(queryParts, "ORDER BY degree DESC")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildBasicStatsQuery builds basic graph statistics query
func (s *Store) buildBasicStatsQuery(graphName string, parameters map[string]interface{}) (string, map[string]interface{}) {
	var queryParts []string

	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(graphName)
		queryParts = append(queryParts, fmt.Sprintf("MATCH (n:%s)", graphLabel))
	} else {
		queryParts = append(queryParts, "MATCH (n)")
	}

	queryParts = append(queryParts, "OPTIONAL MATCH (n)-[r]-()")
	queryParts = append(queryParts, "WITH n, count(r) as degree")
	queryParts = append(queryParts, "RETURN n.id as node_id, degree")

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// buildLeidenQuery builds Leiden community detection query
func (s *Store) buildLeidenQuery(opts *types.CommunityDetectionOptions) (string, map[string]interface{}) {
	// Since Neo4j doesn't have built-in Leiden algorithm, we'll use a simplified approach
	// In practice, you would use APOC procedures or Graph Data Science library
	return s.buildSimpleCommunityQuery(opts, "leiden")
}

// buildLouvainQuery builds Louvain community detection query
func (s *Store) buildLouvainQuery(opts *types.CommunityDetectionOptions) (string, map[string]interface{}) {
	// Since Neo4j doesn't have built-in Louvain algorithm, we'll use a simplified approach
	// In practice, you would use Graph Data Science library
	return s.buildSimpleCommunityQuery(opts, "louvain")
}

// buildLabelPropagationQuery builds Label Propagation community detection query
func (s *Store) buildLabelPropagationQuery(opts *types.CommunityDetectionOptions) (string, map[string]interface{}) {
	// Simplified label propagation - in practice would use GDS procedures
	return s.buildSimpleCommunityQuery(opts, "label_propagation")
}

// buildSimpleCommunityQuery builds a simplified community detection query
func (s *Store) buildSimpleCommunityQuery(opts *types.CommunityDetectionOptions, algorithm string) (string, map[string]interface{}) {
	parameters := make(map[string]interface{})
	var queryParts []string

	// This is a simplified approach - in production you would use Neo4j GDS library
	// For now, we'll group nodes by their connections (basic community detection)

	if !s.useSeparateDatabase {
		graphLabel := s.GetGraphLabel(opts.GraphName)
		queryParts = append(queryParts, fmt.Sprintf("MATCH (n:%s)-[r]-(m:%s)", graphLabel, graphLabel))
	} else {
		queryParts = append(queryParts, "MATCH (n)-[r]-(m)")
	}

	queryParts = append(queryParts, "WITH n, collect(DISTINCT m.id) as neighbors")
	queryParts = append(queryParts, "WITH n, neighbors, size(neighbors) as degree")
	queryParts = append(queryParts, "RETURN n.id as node_id, neighbors, degree")
	queryParts = append(queryParts, "ORDER BY degree DESC")

	// Add limit for max levels if specified
	if opts.MaxLevels > 0 {
		queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", opts.MaxLevels*100)) // Rough estimate
	}

	query := strings.Join(queryParts, " ")
	return query, parameters
}

// parseCommunityResult parses community detection results
func (s *Store) parseCommunityResult(result neo4j.ResultWithContext, opts *types.CommunityDetectionOptions) ([]*types.Community, error) {
	ctx := context.Background()
	var communities []*types.Community
	communityMap := make(map[string]*types.Community)
	communityID := 0

	// Process results and group nodes into communities based on their connections
	for result.Next(ctx) {
		record := result.Record()

		nodeID, _ := record.Get("node_id")
		degree, _ := record.Get("degree")

		nodeIDStr := fmt.Sprintf("%v", nodeID)

		// Simple community assignment based on degree
		// In practice, this would be much more sophisticated
		degreeInt, _ := degree.(int64)
		communityKey := fmt.Sprintf("community_%d", int(degreeInt)%10)

		if community, exists := communityMap[communityKey]; exists {
			// Add to existing community
			community.Members = append(community.Members, nodeIDStr)
			community.Size++
		} else {
			// Create new community
			community := &types.Community{
				ID:      fmt.Sprintf("%s_%d", opts.Algorithm, communityID),
				Level:   0, // Single level for now
				Members: []string{nodeIDStr},
				Size:    1,
				Title:   fmt.Sprintf("Community %d", communityID),
				Summary: fmt.Sprintf("Community detected using %s algorithm", opts.Algorithm),
			}
			communityMap[communityKey] = community
			communities = append(communities, community)
			communityID++
		}
	}

	// Check for query execution errors
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("community detection query error: %w", err)
	}

	return communities, nil
}
