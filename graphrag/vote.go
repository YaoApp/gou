package graphrag

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
)

// StoreKeyVote key format for vote storage (List)
const StoreKeyVote = "segment_votes_%s" // segment_votes_{segmentID}

// StoreKeyVotePositive key format for positive vote count storage
const StoreKeyVotePositive = "segment_positive_%s" // segment_positive_{segmentID}

// StoreKeyVoteNegative key format for negative vote count storage
const StoreKeyVoteNegative = "segment_negative_%s" // segment_negative_{segmentID}

// UpdateVote updates vote for segments
func (g *GraphRag) UpdateVote(ctx context.Context, docID string, segments []types.SegmentVote, options ...types.UpdateVoteOptions) (int, error) {
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
	var wg sync.WaitGroup
	var storeErr, vectorErr error
	updatedCount := 0

	// Update Store concurrently (using List for votes and counters for statistics)
	wg.Add(1)
	go func() {
		defer wg.Done()
		storeUpdated := 0
		for _, segment := range segments {
			// Convert segment vote to map for Store operations
			voteMap, err := segmentVoteToMap(segment)
			if err != nil {
				g.Logger.Warnf("Failed to convert vote to map for segment %s: %v", segment.ID, err)
				continue
			}

			// Add vote to list
			err = g.Store.Push(fmt.Sprintf(StoreKeyVote, segment.ID), voteMap)
			if err != nil {
				g.Logger.Warnf("Failed to add vote for segment %s to Store list: %v", segment.ID, err)
				continue
			}

			// Update statistics counters using tagged switch
			switch segment.Vote {
			case types.VotePositive:
				positiveKey := fmt.Sprintf(StoreKeyVotePositive, segment.ID)
				count, ok := g.Store.Get(positiveKey)
				if !ok {
					count = 0
				}
				if countInt, ok := count.(int); ok {
					err = g.Store.Set(positiveKey, countInt+1, 0)
				} else {
					err = g.Store.Set(positiveKey, 1, 0)
				}
				if err != nil {
					g.Logger.Warnf("Failed to increment positive count for segment %s: %v", segment.ID, err)
				}
			case types.VoteNegative:
				negativeKey := fmt.Sprintf(StoreKeyVoteNegative, segment.ID)
				count, ok := g.Store.Get(negativeKey)
				if !ok {
					count = 0
				}
				if countInt, ok := count.(int); ok {
					err = g.Store.Set(negativeKey, countInt+1, 0)
				} else {
					err = g.Store.Set(negativeKey, 1, 0)
				}
				if err != nil {
					g.Logger.Warnf("Failed to increment negative count for segment %s: %v", segment.ID, err)
				}
			default:
				g.Logger.Warnf("Unknown vote type for segment %s: %v", segment.ID, segment.Vote)
			}

			storeUpdated++
		}
		if storeUpdated < len(segments) {
			storeErr = fmt.Errorf("failed to update some votes in Store: %d/%d updated", storeUpdated, len(segments))
		}
	}()

	// Update Vector DB concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
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
			vectorErr = fmt.Errorf("failed to update vote in vector store: %w", err)
		}
	}()

	wg.Wait()

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

// RemoveVote removes a single vote by VoteID and updates statistics
func (g *GraphRag) RemoveVote(ctx context.Context, docID string, segmentID string, voteID string) error {
	if g.Store == nil {
		return fmt.Errorf("store is not configured, cannot remove vote")
	}

	var wg sync.WaitGroup
	var storeErr, vectorErr error

	// Remove from Store concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get all votes for the segment
		voteKey := fmt.Sprintf(StoreKeyVote, segmentID)
		votes, err := g.Store.ArrayAll(voteKey)
		if err != nil {
			storeErr = fmt.Errorf("failed to get votes from Store: %w", err)
			return
		}

		// Find and remove the vote with matching VoteID
		var removedVote *types.SegmentVote
		var removedVoteMap map[string]interface{}
		for _, v := range votes {
			vote, err := mapToSegmentVote(v)
			if err != nil {
				g.Logger.Warnf("Failed to convert stored vote to struct: %v", err)
				continue
			}
			if vote.VoteID == voteID {
				removedVote = &vote
				// Convert back to map for Pull operation
				removedVoteMap, err = segmentVoteToMap(vote)
				if err != nil {
					storeErr = fmt.Errorf("failed to convert vote to map for removal: %w", err)
					return
				}
				break
			}
		}

		if removedVote == nil {
			storeErr = fmt.Errorf("vote with ID %s not found for segment %s", voteID, segmentID)
			return
		}

		// Remove vote from list using the map representation
		err = g.Store.Pull(voteKey, removedVoteMap)
		if err != nil {
			storeErr = fmt.Errorf("failed to remove vote from Store list: %w", err)
			return
		}

		// Update statistics counters using tagged switch
		switch removedVote.Vote {
		case types.VotePositive:
			positiveKey := fmt.Sprintf(StoreKeyVotePositive, segmentID)
			count, ok := g.Store.Get(positiveKey)
			if ok {
				if countInt, ok := count.(int); ok && countInt > 0 {
					err = g.Store.Set(positiveKey, countInt-1, 0)
					if err != nil {
						g.Logger.Warnf("Failed to decrement positive count for segment %s: %v", segmentID, err)
					}
				}
			}
		case types.VoteNegative:
			negativeKey := fmt.Sprintf(StoreKeyVoteNegative, segmentID)
			count, ok := g.Store.Get(negativeKey)
			if ok {
				if countInt, ok := count.(int); ok && countInt > 0 {
					err = g.Store.Set(negativeKey, countInt-1, 0)
					if err != nil {
						g.Logger.Warnf("Failed to decrement negative count for segment %s: %v", segmentID, err)
					}
				}
			}
		default:
			g.Logger.Warnf("Unknown vote type for segment %s: %v", segmentID, removedVote.Vote)
		}
	}()

	// Update Vector DB metadata concurrently (remove vote metadata)
	wg.Add(1)
	go func() {
		defer wg.Done()

		updates := []segmentMetadataUpdate{
			{
				SegmentID:   segmentID,
				MetadataKey: "vote",
				Value:       nil, // Remove vote metadata
			},
		}

		err := g.updateSegmentMetadataInVectorBatch(ctx, docID, updates)
		if err != nil {
			vectorErr = fmt.Errorf("failed to remove vote in vector store: %w", err)
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
		return fmt.Errorf("failed to remove vote in both Store and Vector DB: Store error: %v, Vector error: %v", storeErr, vectorErr)
	}

	return nil
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

	return g.listVotesForSegment(ctx, options.SegmentID, options)
}

// listVotesForSegment lists votes for a specific segment
func (g *GraphRag) listVotesForSegment(ctx context.Context, segmentID string, options *types.ScrollVotesOptions) (*types.VoteScrollResult, error) {
	voteKey := fmt.Sprintf(StoreKeyVote, segmentID)

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
