package graphrag

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
)

// Backup backs up a collection
func (g *GraphRag) Backup(ctx context.Context, writer io.Writer, id string) error {
	if id == "" {
		return fmt.Errorf("collection ID cannot be empty")
	}

	g.Logger.Infof("Starting backup for collection: %s", id)

	// Use the collection ID directly as graphName
	graphName := id

	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return fmt.Errorf("failed to get collection IDs: %w", err)
	}

	// Create temporary directory for backup files
	tempDir, err := os.MkdirTemp("", "graphrag_backup_*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create backup files concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Backup Vector database
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := g.backupVector(ctx, tempDir, collectionIDs.Vector); err != nil {
			errChan <- fmt.Errorf("vector backup failed: %w", err)
		}
	}()

	// Backup Graph database (if configured)
	if g.Graph != nil && g.Graph.IsConnected() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.backupGraph(ctx, tempDir, graphName); err != nil {
				errChan <- fmt.Errorf("graph backup failed: %w", err)
			}
		}()
	}

	// Backup Store (if configured)
	if g.Store != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.backupStore(ctx, tempDir, graphName); err != nil {
				errChan <- fmt.Errorf("store backup failed: %w", err)
			}
		}()
	}

	// Wait for all backups to complete
	wg.Wait()

	// Check for errors
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Create zip archive
	if err := g.createZipArchive(tempDir, writer); err != nil {
		return fmt.Errorf("failed to create zip archive: %w", err)
	}

	g.Logger.Infof("Backup completed for collection: %s", id)
	return nil
}

// Restore restores a collection
func (g *GraphRag) Restore(ctx context.Context, reader io.Reader, id string) error {
	if id == "" {
		return fmt.Errorf("collection ID cannot be empty")
	}

	g.Logger.Infof("Starting restore for collection: %s", id)

	// Use the collection ID directly as graphName
	graphName := id

	// Get collection IDs
	collectionIDs, err := utils.GetCollectionIDs(graphName)
	if err != nil {
		return fmt.Errorf("failed to get collection IDs: %w", err)
	}

	// Create temporary directory for restore files
	tempDir, err := os.MkdirTemp("", "graphrag_restore_*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract zip archive
	if err := g.extractZipArchive(reader, tempDir); err != nil {
		return fmt.Errorf("failed to extract zip archive: %w", err)
	}

	// Restore from files concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Restore Vector database
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := g.restoreVector(ctx, tempDir, collectionIDs.Vector); err != nil {
			errChan <- fmt.Errorf("vector restore failed: %w", err)
		}
	}()

	// Restore Graph database (if configured)
	if g.Graph != nil && g.Graph.IsConnected() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.restoreGraph(ctx, tempDir, graphName); err != nil {
				errChan <- fmt.Errorf("graph restore failed: %w", err)
			}
		}()
	}

	// Restore Store (if configured)
	if g.Store != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := g.restoreStore(ctx, tempDir, graphName); err != nil {
				errChan <- fmt.Errorf("store restore failed: %w", err)
			}
		}()
	}

	// Wait for all restores to complete
	wg.Wait()

	// Check for errors
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	g.Logger.Infof("Restore completed for collection: %s", id)
	return nil
}

// backupVector backs up vector database to a file
func (g *GraphRag) backupVector(ctx context.Context, tempDir, collectionName string) error {
	if g.Vector == nil {
		g.Logger.Warnf("Vector database not configured, skipping vector backup")
		return nil
	}

	vectorFile := filepath.Join(tempDir, "vector.backup")
	file, err := os.Create(vectorFile)
	if err != nil {
		return fmt.Errorf("failed to create vector backup file: %w", err)
	}
	defer file.Close()

	opts := &types.BackupOptions{
		CollectionName: collectionName,
		Compress:       true,
		ExtraParams: map[string]interface{}{
			"backup_type": "vector",
			"timestamp":   time.Now().Unix(),
		},
	}

	if err := g.Vector.Backup(ctx, file, opts); err != nil {
		return fmt.Errorf("failed to backup vector database: %w", err)
	}

	g.Logger.Debugf("Vector backup completed: %s", vectorFile)
	return nil
}

// backupGraph backs up graph database to a file
func (g *GraphRag) backupGraph(ctx context.Context, tempDir, graphName string) error {
	if g.Graph == nil {
		g.Logger.Warnf("Graph database not configured, skipping graph backup")
		return nil
	}

	graphFile := filepath.Join(tempDir, "graph.backup")
	file, err := os.Create(graphFile)
	if err != nil {
		return fmt.Errorf("failed to create graph backup file: %w", err)
	}
	defer file.Close()

	opts := &types.GraphBackupOptions{
		GraphName: graphName,
		Format:    "json",
		Compress:  true,
		ExtraParams: map[string]interface{}{
			"backup_type": "graph",
			"timestamp":   time.Now().Unix(),
		},
	}

	if err := g.Graph.Backup(ctx, file, opts); err != nil {
		return fmt.Errorf("failed to backup graph database: %w", err)
	}

	g.Logger.Debugf("Graph backup completed: %s", graphFile)
	return nil
}

// backupStore backs up store database to files
func (g *GraphRag) backupStore(ctx context.Context, tempDir, graphName string) error {
	if g.Store == nil {
		g.Logger.Warnf("Store database not configured, skipping store backup")
		return nil
	}

	storeDir := filepath.Join(tempDir, "store")
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return fmt.Errorf("failed to create store backup directory: %w", err)
	}

	// Get all keys from the store
	allKeys := g.Store.Keys()

	// Filter keys that match our prefix pattern
	prefix := fmt.Sprintf("graphrag:%s:", graphName)
	var keys []string
	for _, key := range allKeys {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	// Create store manifest file
	manifestFile := filepath.Join(storeDir, "manifest.json")
	manifest := map[string]interface{}{
		"backup_type": "store",
		"timestamp":   time.Now().Unix(),
		"graph_name":  graphName,
		"key_count":   len(keys),
		"keys":        keys,
	}

	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal store manifest: %w", err)
	}

	if err := os.WriteFile(manifestFile, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write store manifest: %w", err)
	}

	// Backup each key's value to a separate file
	for i, key := range keys {
		value, ok := g.Store.Get(key)
		if !ok {
			g.Logger.Warnf("Failed to get value for key %s", key)
			continue
		}

		keyFile := filepath.Join(storeDir, fmt.Sprintf("key_%d.data", i))
		keyData := map[string]interface{}{
			"key":   key,
			"value": value,
		}

		keyDataBytes, err := json.Marshal(keyData)
		if err != nil {
			g.Logger.Warnf("Failed to marshal data for key %s: %v", key, err)
			continue
		}

		if err := os.WriteFile(keyFile, keyDataBytes, 0644); err != nil {
			g.Logger.Warnf("Failed to write data for key %s: %v", key, err)
			continue
		}
	}

	g.Logger.Debugf("Store backup completed: %s (%d keys)", storeDir, len(keys))
	return nil
}

// restoreVector restores vector database from a file
func (g *GraphRag) restoreVector(ctx context.Context, tempDir, collectionName string) error {
	if g.Vector == nil {
		g.Logger.Warnf("Vector database not configured, skipping vector restore")
		return nil
	}

	vectorFile := filepath.Join(tempDir, "vector.backup")
	if _, err := os.Stat(vectorFile); os.IsNotExist(err) {
		g.Logger.Warnf("Vector backup file not found, skipping vector restore")
		return nil
	}

	file, err := os.Open(vectorFile)
	if err != nil {
		return fmt.Errorf("failed to open vector backup file: %w", err)
	}
	defer file.Close()

	opts := &types.RestoreOptions{
		CollectionName: collectionName,
		Force:          true,
		ExtraParams: map[string]interface{}{
			"restore_type": "vector",
		},
	}

	if err := g.Vector.Restore(ctx, file, opts); err != nil {
		return fmt.Errorf("failed to restore vector database: %w", err)
	}

	g.Logger.Debugf("Vector restore completed: %s", vectorFile)
	return nil
}

// restoreGraph restores graph database from a file
func (g *GraphRag) restoreGraph(ctx context.Context, tempDir, graphName string) error {
	if g.Graph == nil {
		g.Logger.Warnf("Graph database not configured, skipping graph restore")
		return nil
	}

	graphFile := filepath.Join(tempDir, "graph.backup")
	if _, err := os.Stat(graphFile); os.IsNotExist(err) {
		g.Logger.Warnf("Graph backup file not found, skipping graph restore")
		return nil
	}

	file, err := os.Open(graphFile)
	if err != nil {
		return fmt.Errorf("failed to open graph backup file: %w", err)
	}
	defer file.Close()

	opts := &types.GraphRestoreOptions{
		GraphName:   graphName,
		Format:      "json",
		Force:       true,
		CreateGraph: true,
		ExtraParams: map[string]interface{}{
			"restore_type": "graph",
		},
	}

	if err := g.Graph.Restore(ctx, file, opts); err != nil {
		return fmt.Errorf("failed to restore graph database: %w", err)
	}

	g.Logger.Debugf("Graph restore completed: %s", graphFile)
	return nil
}

// restoreStore restores store database from files
func (g *GraphRag) restoreStore(ctx context.Context, tempDir, graphName string) error {
	if g.Store == nil {
		g.Logger.Warnf("Store database not configured, skipping store restore")
		return nil
	}

	storeDir := filepath.Join(tempDir, "store")
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		g.Logger.Warnf("Store backup directory not found, skipping store restore")
		return nil
	}

	// Read manifest file
	manifestFile := filepath.Join(storeDir, "manifest.json")
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		return fmt.Errorf("store manifest file not found: %s", manifestFile)
	}

	manifestData, err := os.ReadFile(manifestFile)
	if err != nil {
		return fmt.Errorf("failed to read store manifest: %w", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal store manifest: %w", err)
	}

	// Restore each key's value from files
	keyCount := int(manifest["key_count"].(float64))
	restoredCount := 0

	for i := 0; i < keyCount; i++ {
		keyFile := filepath.Join(storeDir, fmt.Sprintf("key_%d.data", i))
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			continue
		}

		keyDataBytes, err := os.ReadFile(keyFile)
		if err != nil {
			g.Logger.Warnf("Failed to read key file %s: %v", keyFile, err)
			continue
		}

		var keyData map[string]interface{}
		if err := json.Unmarshal(keyDataBytes, &keyData); err != nil {
			g.Logger.Warnf("Failed to unmarshal key data from %s: %v", keyFile, err)
			continue
		}

		key, ok := keyData["key"].(string)
		if !ok {
			g.Logger.Warnf("Invalid key format in %s", keyFile)
			continue
		}

		value := keyData["value"]

		if err := g.Store.Set(key, value, 0); err != nil {
			g.Logger.Warnf("Failed to set value for key %s: %v", key, err)
			continue
		}

		restoredCount++
	}

	g.Logger.Debugf("Store restore completed: %s (%d/%d keys restored)", storeDir, restoredCount, keyCount)
	return nil
}

// createZipArchive creates a zip archive from the temporary directory
func (g *GraphRag) createZipArchive(tempDir string, writer io.Writer) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		// Create file in zip
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		return err
	})
}

// extractZipArchive extracts a zip archive to the temporary directory
func (g *GraphRag) extractZipArchive(reader io.Reader, tempDir string) error {
	// Read all data to memory for zip.NewReader
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read zip data: %w", err)
	}

	// Create zip reader
	zipReader, err := zip.NewReader(&readerAt{data}, int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	// Extract files
	for _, file := range zipReader.File {
		path := filepath.Join(tempDir, file.Name)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Create file
		outFile, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		// Open file in zip
		fileReader, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		// Copy content
		_, err = io.Copy(outFile, fileReader)
		fileReader.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	return nil
}

// readerAt is a helper type to implement io.ReaderAt from []byte
type readerAt struct {
	data []byte
}

func (r *readerAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	n = copy(p, r.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
