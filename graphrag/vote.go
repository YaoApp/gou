package graphrag

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
)

// StoreKeyVote key format for vote storage (List)
const StoreKeyVote = "doc:%s:segment:votes:%s" // doc:{docID}:segment:votes:{segmentID}

// StoreKeyVotePositive key format for positive vote count storage
const StoreKeyVotePositive = "doc:%s:segment:positive:%s" // doc:{docID}:segment:positive:{segmentID}

// StoreKeyVoteNegative key format for negative vote count storage
const StoreKeyVoteNegative = "doc:%s:segment:negative:%s" // doc:{docID}:segment:negative:{segmentID}

// UpdateVotes updates vote for segments
func (g *GraphRag) UpdateVotes(ctx context.Context, docID string, segments []types.SegmentVote, options ...types.UpdateVoteOptions) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}

	// Apply reaction from options if SegmentReaction is not provided in segments
	if len(options) > 0 && options[0].Reaction != nil {
		for i := range segments {
			if segments[i].SegmentReaction == nil {
				segments[i].SegmentReaction = options[0].Reaction
			}
		}
	}

	// Generate VoteIDs for segments that don't have them
	for i := range segments {
		if segments[i].VoteID == "" {
			segments[i].VoteID = uuid.New().String()
		}
	}

	// Compute vote if compute is provided
	if len(options) > 0 && options[0].Compute != nil {
		segmentIDs := make([]string, len(segments))
		for i, segment := range segments {
			segmentIDs[i] = segment.ID
		}

		var context map[string]interface{}
		if len(options) > 0 && options[0].Reaction != nil {
			context = options[0].Reaction.Context
		}

		votes, err := options[0].Compute.Compute(ctx, docID, segmentIDs, context, options[0].Progress)
		if err != nil {
			return 0, fmt.Errorf("failed to compute votes for segments: %w", err)
		}

		if len(votes) != len(segments) {
			return 0, fmt.Errorf("compute returned %d votes but expected %d", len(votes), len(segments))
		}

		for i, vote := range votes {
			segments[i].Vote = vote
		}
	}

	// Strategy 1: Store not configured - use Vector DB metadata only
	if g.Store == nil {
		return g.updateVoteInVectorOnly(ctx, docID, segments)
	}

	// Strategy 2: Store configured - concurrent update to Store and Vector DB
	return g.updateVoteInStoreAndVector(ctx, docID, segments)
}

// updateVoteInVectorOnly updates votes in Vector DB metadata only
func (g *GraphRag) updateVoteInVectorOnly(ctx context.Context, docID string, segments []types.SegmentVote) (int, error) {
	var updates []segmentMetadataUpdate
	for _, segment := range segments {
		updates = append(updates, segmentMetadataUpdate{
			SegmentID:   segment.ID,
			MetadataKey: "vote",
			Value:       segment.Vote,
		})
	}

	err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
	if err != nil {
		return 0, fmt.Errorf("failed to update vote in vector store: %w", err)
	}

	return len(segments), nil
}

// updateVoteInStoreAndVector updates votes in both Store (as List) and Vector DB
func (g *GraphRag) updateVoteInStoreAndVector(ctx context.Context, docID string, segments []types.SegmentVote) (int, error) {
	var storeErr, vectorErr error
	updatedCount := 0

	// Group segments by ID for processing
	segmentsByID := make(map[string][]types.SegmentVote)
	for _, segment := range segments {
		segmentsByID[segment.ID] = append(segmentsByID[segment.ID], segment)
	}

	// Process each unique segment ID
	finalVoteCounts := make(map[string]map[string]int) // segmentID -> {positive: count, negative: count}
	storeUpdated := 0

	for segmentID, segmentVotes := range segmentsByID {
		voteKey := fmt.Sprintf(StoreKeyVote, docID, segmentID)
		positiveKey := fmt.Sprintf(StoreKeyVotePositive, docID, segmentID)
		negativeKey := fmt.Sprintf(StoreKeyVoteNegative, docID, segmentID)

		// Add all votes to the list first
		for _, segment := range segmentVotes {
			voteMap, err := segmentVoteToMap(segment)
			if err != nil {
				g.Logger.Warnf("Failed to convert vote to map for segment %s: %v", segment.ID, err)
				continue
			}

			err = g.Store.Push(voteKey, voteMap)
			if err != nil {
				g.Logger.Warnf("Failed to add vote for segment %s to Store list: %v", segment.ID, err)
				continue
			}
			storeUpdated++
		}

		// Calculate accurate counts directly from the list
		actualCounts := g.calculateVoteCountsFromList(voteKey)

		// Update the count caches to match actual records
		err := g.Store.Set(positiveKey, actualCounts["positive"], 0)
		if err != nil {
			g.Logger.Warnf("Failed to update positive count for segment %s: %v", segmentID, err)
		}

		err = g.Store.Set(negativeKey, actualCounts["negative"], 0)
		if err != nil {
			g.Logger.Warnf("Failed to update negative count for segment %s: %v", segmentID, err)
		}

		// Store the final counts for Vector DB update
		finalVoteCounts[segmentID] = actualCounts
	}

	if storeUpdated < len(segments) {
		storeErr = fmt.Errorf("failed to update some votes in Store: %d/%d updated", storeUpdated, len(segments))
	}

	// Step 2: Update Vector DB with the accurate counts from Store
	if len(finalVoteCounts) > 0 {
		var updates []segmentMetadataUpdate
		for segmentID, counts := range finalVoteCounts {
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segmentID,
				MetadataKey: "positive",
				Value:       counts["positive"],
			})
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segmentID,
				MetadataKey: "negative",
				Value:       counts["negative"],
			})
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to update vote in vector store: %w", err)
		}
	}

	// Count successful updates (at least one storage succeeded)
	if storeErr == nil || vectorErr == nil {
		updatedCount = len(segments)
	}

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store update error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB update error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to update vote in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return updatedCount, nil
}

// calculateVoteCountsFromList calculates vote counts by reading all votes from the list
func (g *GraphRag) calculateVoteCountsFromList(voteKey string) map[string]int {
	counts := map[string]int{"positive": 0, "negative": 0}

	allVotes, err := g.Store.ArrayAll(voteKey)
	if err != nil {
		g.Logger.Warnf("Failed to get votes from Store for counting: %v", err)
		return counts
	}

	for _, v := range allVotes {
		vote, err := mapToSegmentVote(v)
		if err != nil {
			g.Logger.Warnf("Failed to convert stored vote to struct for counting: %v", err)
			continue
		}

		switch vote.Vote {
		case types.VotePositive:
			counts["positive"]++
		case types.VoteNegative:
			counts["negative"]++
		}
	}

	return counts
}

// RemoveVotes removes multiple votes by VoteID and updates statistics
func (g *GraphRag) RemoveVotes(ctx context.Context, docID string, votes []types.VoteRemoval) (int, error) {
	if len(votes) == 0 {
		return 0, nil
	}

	if g.Store == nil {
		return 0, fmt.Errorf("store is not configured, cannot remove votes")
	}

	var wg sync.WaitGroup
	var storeErr, vectorErr error
	removedCount := 0

	// Group votes by segment ID for efficient processing
	votesBySegment := make(map[string][]types.VoteRemoval)
	for _, vote := range votes {
		votesBySegment[vote.SegmentID] = append(votesBySegment[vote.SegmentID], vote)
	}

	// Remove from Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeRemoved := 0

		for segmentID, segmentVotes := range votesBySegment {
			// Get all votes for the segment
			voteKey := fmt.Sprintf(StoreKeyVote, docID, segmentID)
			allVotes, err := g.Store.ArrayAll(voteKey)
			if err != nil {
				g.Logger.Warnf("Failed to get votes from Store for segment %s: %v", segmentID, err)
				continue
			}

			// Create a map of VoteID to remove for efficient lookup
			votesToRemove := make(map[string]bool)
			for _, v := range segmentVotes {
				votesToRemove[v.VoteID] = true
			}

			// Find votes to remove
			var removedVotes []types.SegmentVote
			var votesToKeep []interface{}

			for _, v := range allVotes {
				vote, err := mapToSegmentVote(v)
				if err != nil {
					g.Logger.Warnf("Failed to convert stored vote to struct: %v", err)
					votesToKeep = append(votesToKeep, v) // Keep invalid votes
					continue
				}

				if votesToRemove[vote.VoteID] {
					// This vote should be removed
					removedVotes = append(removedVotes, vote)
				} else {
					// Keep this vote
					votesToKeep = append(votesToKeep, v)
				}
			}

			// Update the vote list
			if len(removedVotes) > 0 {
				// Clear the list and re-add remaining votes
				g.Store.Del(voteKey)
				if len(votesToKeep) > 0 {
					err = g.Store.Push(voteKey, votesToKeep...)
				}
				if err != nil {
					g.Logger.Warnf("Failed to update vote list for segment %s: %v", segmentID, err)
					continue
				}

				// Update vote counts to match actual records after removal
				positiveKey := fmt.Sprintf(StoreKeyVotePositive, docID, segmentID)
				negativeKey := fmt.Sprintf(StoreKeyVoteNegative, docID, segmentID)
				actualCounts := g.calculateVoteCountsFromList(voteKey)

				if actualCounts["positive"] == 0 {
					g.Store.Del(positiveKey)
				} else {
					g.Store.Set(positiveKey, actualCounts["positive"], 0)
				}

				if actualCounts["negative"] == 0 {
					g.Store.Del(negativeKey)
				} else {
					g.Store.Set(negativeKey, actualCounts["negative"], 0)
				}

				storeRemoved += len(removedVotes)
			}
		}

		if storeRemoved < len(votes) {
			storeErr = fmt.Errorf("failed to remove some votes in Store: %d/%d removed", storeRemoved, len(votes))
		}
	}()

	// Update Vector DB metadata concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Group votes by segment ID
		segmentIDs := make(map[string]bool)
		for _, vote := range votes {
			segmentIDs[vote.SegmentID] = true
		}

		var updates []segmentMetadataUpdate
		for segmentID := range segmentIDs {
			// Get actual counts from Store after removal
			voteKey := fmt.Sprintf(StoreKeyVote, docID, segmentID)
			actualCounts := g.calculateVoteCountsFromList(voteKey)

			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segmentID,
				MetadataKey: "positive",
				Value:       actualCounts["positive"], // Use actual count from Store
			})
			updates = append(updates, segmentMetadataUpdate{
				SegmentID:   segmentID,
				MetadataKey: "negative",
				Value:       actualCounts["negative"], // Use actual count from Store
			})
		}

		if len(updates) > 0 {
			err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
			if err != nil {
				vectorErr = fmt.Errorf("failed to update votes in vector store: %w", err)
			}
		}
	}()

	wg.Wait()

	// Count successful removals (at least one storage succeeded)
	if storeErr == nil || vectorErr == nil {
		removedCount = len(votes)
	}

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store remove error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB remove error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to remove votes in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return removedCount, nil
}

// RemoveVotesBySegmentID removes all votes for a segment and clears statistics
func (g *GraphRag) RemoveVotesBySegmentID(ctx context.Context, docID string, segmentID string) (int, error) {
	if g.Store == nil {
		return 0, fmt.Errorf("store is not configured, cannot remove votes")
	}

	var wg sync.WaitGroup
	var storeErr, vectorErr error
	removedCount := 0

	// Remove from Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get all votes for the segment
		voteKey := fmt.Sprintf(StoreKeyVote, docID, segmentID)
		allVotes, err := g.Store.ArrayAll(voteKey)
		if err != nil {
			g.Logger.Warnf("Failed to get votes from Store for segment %s: %v", segmentID, err)
			storeErr = fmt.Errorf("failed to get votes from Store: %w", err)
			return
		}

		// Count votes before removal
		removedCount = len(allVotes)

		// Clear all votes for the segment
		g.Store.Del(voteKey)

		// Clear statistics counters
		positiveKey := fmt.Sprintf(StoreKeyVotePositive, docID, segmentID)
		negativeKey := fmt.Sprintf(StoreKeyVoteNegative, docID, segmentID)
		g.Store.Del(positiveKey)
		g.Store.Del(negativeKey)
	}()

	// Update Vector DB metadata concurrently (clear vote counts)
	wg.Add(1)
	go func() {
		defer wg.Done()

		updates := []segmentMetadataUpdate{
			{
				SegmentID:   segmentID,
				MetadataKey: "positive",
				Value:       0, // Clear positive count to 0
			},
			{
				SegmentID:   segmentID,
				MetadataKey: "negative",
				Value:       0, // Clear negative count to 0
			},
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to remove votes in vector store: %w", err)
		}
	}()

	wg.Wait()

	// Log any errors but don't fail completely if one storage succeeded
	if storeErr != nil {
		g.Logger.Warnf("Store remove error: %v", storeErr)
	}
	if vectorErr != nil {
		g.Logger.Warnf("Vector DB remove error: %v", vectorErr)
	}

	// Return error only if both failed
	if storeErr != nil && vectorErr != nil {
		return 0, fmt.Errorf("failed to remove votes in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return removedCount, nil
}

// ScrollVotes scrolls votes for a document with pagination support
func (g *GraphRag) ScrollVotes(ctx context.Context, docID string, options *types.ScrollVotesOptions) (*types.VoteScrollResult, error) {
	if g.Store == nil {
		return nil, fmt.Errorf("store is not configured, cannot list votes")
	}

	if options == nil {
		options = &types.ScrollVotesOptions{}
	}

	// Set default limit
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Limit > 100 {
		options.Limit = 100
	}

	// SegmentID is required for listing votes
	if options.SegmentID == "" {
		return nil, fmt.Errorf("segment_id is required for listing votes")
	}

	return g.listVotesForSegment(ctx, docID, options.SegmentID, options)
}

// listVotesForSegment lists votes for a specific segment
func (g *GraphRag) listVotesForSegment(ctx context.Context, docID string, segmentID string, options *types.ScrollVotesOptions) (*types.VoteScrollResult, error) {
	voteKey := fmt.Sprintf(StoreKeyVote, docID, segmentID)

	// Get all votes for the segment
	allVotes, err := g.Store.ArrayAll(voteKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes from Store: %w", err)
	}

	// Convert to SegmentVote slice and apply filters
	var votes []types.SegmentVote
	for _, v := range allVotes {
		vote, err := mapToSegmentVote(v)
		if err != nil {
			g.Logger.Warnf("Failed to convert stored vote to struct: %v", err)
			continue
		}

		// Apply filters
		if !g.matchesVoteFilters(vote, options) {
			continue
		}

		votes = append(votes, vote)
	}

	return g.paginateVotes(votes, options)
}

// matchesVoteFilters checks if a vote matches the filter criteria
func (g *GraphRag) matchesVoteFilters(vote types.SegmentVote, options *types.ScrollVotesOptions) bool {
	// Filter by vote type
	if options.VoteType != "" && vote.Vote != options.VoteType {
		return false
	}

	// Filter by reaction source
	if options.Source != "" && vote.SegmentReaction != nil && vote.SegmentReaction.Source != options.Source {
		return false
	}

	// Filter by reaction scenario
	if options.Scenario != "" && vote.SegmentReaction != nil && vote.SegmentReaction.Scenario != options.Scenario {
		return false
	}

	return true
}

// paginateVotes handles pagination of vote results
func (g *GraphRag) paginateVotes(votes []types.SegmentVote, options *types.ScrollVotesOptions) (*types.VoteScrollResult, error) {
	result := &types.VoteScrollResult{
		Total: len(votes),
	}

	// Find start index based on cursor
	startIndex := 0
	if options.Cursor != "" {
		// Find the vote with the cursor VoteID
		for i, vote := range votes {
			if vote.VoteID == options.Cursor {
				startIndex = i + 1
				break
			}
		}
	}

	// Calculate end index
	endIndex := startIndex + options.Limit
	if endIndex > len(votes) {
		endIndex = len(votes)
	}

	// Extract the page of votes
	if startIndex < len(votes) {
		result.Votes = votes[startIndex:endIndex]
	}

	// Set HasMore and NextCursor
	if endIndex < len(votes) {
		result.HasMore = true
		if len(result.Votes) > 0 {
			result.NextCursor = result.Votes[len(result.Votes)-1].VoteID
		}
	}

	return result, nil
}

// GetVote gets a single vote by ID
func (g *GraphRag) GetVote(ctx context.Context, docID string, segmentID string, voteID string) (*types.SegmentVote, error) {
	if g.Store == nil {
		return nil, fmt.Errorf("store is not configured, cannot get vote")
	}

	// Get all votes for the segment
	voteKey := fmt.Sprintf(StoreKeyVote, docID, segmentID)
	allVotes, err := g.Store.ArrayAll(voteKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes from Store: %w", err)
	}

	// Find the specific vote by voteID
	for _, v := range allVotes {
		vote, err := mapToSegmentVote(v)
		if err != nil {
			g.Logger.Warnf("Failed to convert stored vote to struct: %v", err)
			continue
		}

		if vote.VoteID == voteID {
			return &vote, nil
		}
	}

	return nil, fmt.Errorf("vote not found")
}
