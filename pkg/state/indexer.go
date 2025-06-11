package state

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type BlockIndex struct {
	Height           uint64    `json:"height"`
	Hash             string    `json:"hash"`
	Timestamp        time.Time `json:"timestamp"`
	CelestiaHeight   uint64    `json:"celestia_height"`
	ParentHash       string    `json:"parent_hash"`
	DirsRoot         string    `json:"dirs_root"`
	FilesRoot        string    `json:"files_root"`
	ChunksRoot       string    `json:"chunks_root"`
	StateRoot        string    `json:"state_root"`
	TotalChunks      int       `json:"total_chunks"`
	TotalFiles       int       `json:"total_files"`
	TotalDirectories int       `json:"total_directories"`
	StorageUsed      uint64    `json:"storage_used"`
}

type ChunkIndex struct {
	BlobID      string `json:"blob_id"`
	BlockHeight uint64 `json:"block_height"`
	Index       int    `json:"index"`
	ChunkSize   uint64 `json:"chunk_size"`
	ChunkHash   string `json:"chunk_hash"`
}

type FileIndex struct {
	BlobID              string   `json:"blob_id"`
	FileName            string   `json:"file_name"`
	MimeType            string   `json:"mime_type"`
	FileSize            uint64   `json:"file_size"`
	FileHash            string   `json:"file_hash"`
	BlockHeight         uint64   `json:"block_height"`
	ChunkCount          int      `json:"chunk_count"`
	DirectoryReferences []string `json:"directory_references"`
	Tags                []string `json:"tags"`
}

type DirectoryIndex struct {
	BlobID        string   `json:"blob_id"`
	DirectoryName string   `json:"directory_name"`
	DirectoryHash string   `json:"directory_hash"`
	BlockHeight   uint64   `json:"block_height"`
	FileCount     int      `json:"file_count"`
	TotalSize     uint64   `json:"total_size"`
	FileTypes     []string `json:"file_types"`
	SubPaths      []string `json:"sub_paths"`
}

type SearchResult struct {
	Type        string    `json:"type"` // "file", "directory", "block"
	BlobID      string    `json:"blob_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	BlockHeight uint64    `json:"block_height"`
	Timestamp   time.Time `json:"timestamp"`
	MatchScore  float64   `json:"match_score"`
}

type FileFilters struct {
	MimeType string
	MinSize  uint64
	MaxSize  uint64
	After    *time.Time
	Before   *time.Time
}

type StorageAnalytics struct {
	TotalBlocks          uint64            `json:"total_blocks"`
	TotalChunks          uint64            `json:"total_chunks"`
	TotalFiles           uint64            `json:"total_files"`
	TotalDirectories     uint64            `json:"total_directories"`
	TotalStorage         uint64            `json:"total_storage"`
	AvgBlockSize         uint64            `json:"avg_block_size"`
	MostCommonMimeTypes  map[string]int    `json:"most_common_mime_types"`
	FileTypeDistribution map[string]uint64 `json:"file_type_distribution"`
	LargestFiles         []*FileIndex      `json:"largest_files"`
	LargestDirectories   []*DirectoryIndex `json:"largest_directories"`
}

type IndexerDatabase struct {
	db *sql.DB
}

func NewIndexerDatabase(indexerDbPath string) (*IndexerDatabase, error) {
	if indexerDbPath == "" {
		var err error
		indexerDbPath, err = getStateDbPath(IndexerStateDb)
		if err != nil {
			return nil, fmt.Errorf("error getting indexer database path: %v", err)
		}
	}
	slog.Info("opening indexer database", "path", indexerDbPath)

	// Open with WAL mode for better concurrent access
	db, err := sql.Open("sqlite3", indexerDbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("error opening indexer database: %v", err)
	}

	idb := &IndexerDatabase{db: db}
	if err := idb.initSchema(); err != nil {
		return nil, fmt.Errorf("error initializing database schema: %v", err)
	}

	return idb, nil
}

func (idb *IndexerDatabase) Close() error {
	return idb.db.Close()
}

func (idb *IndexerDatabase) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS blocks (
		height INTEGER PRIMARY KEY,
		version INTEGER NOT NULL DEFAULT 1,
		chain_id TEXT NOT NULL DEFAULT '',
		hash TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		celestia_height INTEGER NOT NULL,
		parent_hash TEXT NOT NULL DEFAULT '',
		dirs_root TEXT NOT NULL DEFAULT '',
		files_root TEXT NOT NULL DEFAULT '',
		chunks_root TEXT NOT NULL DEFAULT '',
		state_root TEXT NOT NULL DEFAULT '',
		total_chunks INTEGER NOT NULL,
		total_files INTEGER NOT NULL,
		total_directories INTEGER NOT NULL,
		storage_used INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS files (
		blob_id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		mime_type TEXT,
		file_size INTEGER NOT NULL,
		file_hash TEXT NOT NULL,
		block_height INTEGER NOT NULL,
		chunk_count INTEGER NOT NULL,
		tags TEXT, -- JSON array as string
		FOREIGN KEY (block_height) REFERENCES blocks(height)
	);

	CREATE TABLE IF NOT EXISTS directories (
		blob_id TEXT PRIMARY KEY,
		directory_name TEXT NOT NULL,
		directory_hash TEXT NOT NULL,
		block_height INTEGER NOT NULL,
		file_count INTEGER NOT NULL,
		total_size INTEGER NOT NULL,
		file_types TEXT, -- JSON array as string
		sub_paths TEXT,  -- JSON array as string
		FOREIGN KEY (block_height) REFERENCES blocks(height)
	);

	CREATE TABLE IF NOT EXISTS directory_files (
		directory_id TEXT NOT NULL,
		file_id TEXT NOT NULL,
		relative_path TEXT NOT NULL,
		FOREIGN KEY (directory_id) REFERENCES directories(blob_id),
		FOREIGN KEY (file_id) REFERENCES files(blob_id)
	);

	CREATE TABLE IF NOT EXISTS chunks (
		blob_id TEXT PRIMARY KEY,
		block_height INTEGER NOT NULL,
		chunk_index INTEGER NOT NULL,
		chunk_size INTEGER NOT NULL,
		chunk_hash TEXT NOT NULL,
		FOREIGN KEY (block_height) REFERENCES blocks(height)
	);

	CREATE TABLE IF NOT EXISTS file_chunks (
		file_id TEXT NOT NULL,
		chunk_id TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		FOREIGN KEY (file_id) REFERENCES files(blob_id),
		FOREIGN KEY (chunk_id) REFERENCES chunks(blob_id)
	);

	-- Indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_files_mime_type ON files(mime_type);
	CREATE INDEX IF NOT EXISTS idx_files_block_height ON files(block_height);
	CREATE INDEX IF NOT EXISTS idx_files_file_size ON files(file_size);
	CREATE INDEX IF NOT EXISTS idx_directories_block_height ON directories(block_height);
	CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks(timestamp);
	CREATE INDEX IF NOT EXISTS idx_chunks_block_height ON chunks(block_height);

	-- Full-text search tables
	CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
		filename,
		content='files',
		content_rowid='rowid'
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS directories_fts USING fts5(
		directory_name,
		content='directories',
		content_rowid='rowid'
	);

	-- Triggers to keep FTS tables in sync
	CREATE TRIGGER IF NOT EXISTS files_fts_insert AFTER INSERT ON files BEGIN
		INSERT INTO files_fts(rowid, filename) VALUES (new.rowid, new.filename);
	END;

	CREATE TRIGGER IF NOT EXISTS files_fts_delete AFTER DELETE ON files BEGIN
		DELETE FROM files_fts WHERE rowid = old.rowid;
	END;

	CREATE TRIGGER IF NOT EXISTS files_fts_update AFTER UPDATE ON files BEGIN
		DELETE FROM files_fts WHERE rowid = old.rowid;
		INSERT INTO files_fts(rowid, filename) VALUES (new.rowid, new.filename);
	END;

	CREATE TRIGGER IF NOT EXISTS directories_fts_insert AFTER INSERT ON directories BEGIN
		INSERT INTO directories_fts(rowid, directory_name) VALUES (new.rowid, new.directory_name);
	END;

	CREATE TRIGGER IF NOT EXISTS directories_fts_delete AFTER DELETE ON directories BEGIN
		DELETE FROM directories_fts WHERE rowid = old.rowid;
	END;

	CREATE TRIGGER IF NOT EXISTS directories_fts_update AFTER UPDATE ON directories BEGIN
		DELETE FROM directories_fts WHERE rowid = old.rowid;
		INSERT INTO directories_fts(rowid, directory_name) VALUES (new.rowid, new.directory_name);
	END;
	`

	_, err := idb.db.Exec(schema)
	return err
}

func (idb *IndexerDatabase) PutBlockIndex(block *BlockIndex) error {
	query := `
	INSERT OR REPLACE INTO blocks 
	(height, hash, timestamp, celestia_height, total_chunks, total_files, total_directories, storage_used, parent_hash, dirs_root, files_root, chunks_root, state_root)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := idb.db.Exec(query,
		block.Height,
		block.Hash,
		block.Timestamp,
		block.CelestiaHeight,
		block.TotalChunks,
		block.TotalFiles,
		block.TotalDirectories,
		block.StorageUsed,
		block.ParentHash,
		block.DirsRoot,
		block.FilesRoot,
		block.ChunksRoot,
		block.StateRoot,
	)

	return err
}

func (idb *IndexerDatabase) GetBlockIndex(height uint64) (*BlockIndex, error) {
	query := `
	SELECT height, hash, timestamp, celestia_height, total_chunks, total_files, total_directories, storage_used, parent_hash, dirs_root, files_root, chunks_root, state_root
	FROM blocks 
	WHERE height = ?
	`

	var block BlockIndex
	err := idb.db.QueryRow(query, height).Scan(
		&block.Height,
		&block.Hash,
		&block.Timestamp,
		&block.CelestiaHeight,
		&block.TotalChunks,
		&block.TotalFiles,
		&block.TotalDirectories,
		&block.StorageUsed,
		&block.ParentHash,
		&block.DirsRoot,
		&block.FilesRoot,
		&block.ChunksRoot,
		&block.StateRoot,
	)

	if err != nil {
		return nil, err
	}

	return &block, nil
}

func (idb *IndexerDatabase) PutFileIndex(file *FileIndex) error {
	query := `
	INSERT OR REPLACE INTO files 
	(blob_id, filename, mime_type, file_size, file_hash, block_height, chunk_count, tags)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Convert tags to JSON string (simple implementation)
	tagsJSON := strings.Join(file.Tags, ",")

	_, err := idb.db.Exec(query,
		file.BlobID,
		file.FileName,
		file.MimeType,
		file.FileSize,
		file.FileHash,
		file.BlockHeight,
		file.ChunkCount,
		tagsJSON,
	)

	return err
}

func (idb *IndexerDatabase) PutFileChunk(fileID string, chunkID string, chunkIndex int) error {
	query := `
	INSERT OR REPLACE INTO file_chunks (file_id, chunk_id, chunk_index)
	VALUES (?, ?, ?)
	`

	_, err := idb.db.Exec(query, fileID, chunkID, chunkIndex)
	return err
}

func (idb *IndexerDatabase) GetFileIndex(blobID string) (*FileIndex, error) {
	query := `
	SELECT blob_id, filename, mime_type, file_size, file_hash, block_height, chunk_count, tags
	FROM files 
	WHERE blob_id = ?
	`

	var file FileIndex
	var tagsJSON string

	err := idb.db.QueryRow(query, blobID).Scan(
		&file.BlobID,
		&file.FileName,
		&file.MimeType,
		&file.FileSize,
		&file.FileHash,
		&file.BlockHeight,
		&file.ChunkCount,
		&tagsJSON,
	)

	if err != nil {
		return nil, err
	}

	// Convert tags back from JSON
	if tagsJSON != "" {
		file.Tags = strings.Split(tagsJSON, ",")
	}

	return &file, nil
}

func (idb *IndexerDatabase) GetFileChunks(fileID string) ([]*ChunkIndex, error) {
	query := `
	SELECT chunks.blob_id, file_chunks.chunk_index, chunks.block_height, chunks.chunk_size, chunks.chunk_hash
	FROM file_chunks
	JOIN chunks ON file_chunks.chunk_id = chunks.blob_id
	WHERE file_id = ?
	ORDER BY file_chunks.chunk_index ASC
	`

	rows, err := idb.db.Query(query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*ChunkIndex
	for rows.Next() {
		var chunk ChunkIndex
		err := rows.Scan(&chunk.BlobID, &chunk.Index, &chunk.BlockHeight, &chunk.ChunkSize, &chunk.ChunkHash)
		if err != nil {
			continue
		}
		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}

func (idb *IndexerDatabase) PutDirectoryIndex(dir *DirectoryIndex) error {
	query := `
	INSERT OR REPLACE INTO directories 
	(blob_id, directory_name, directory_hash, block_height, file_count, total_size, file_types, sub_paths)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Convert slices to JSON strings (simple implementation)
	fileTypesJSON := strings.Join(dir.FileTypes, ",")
	subPathsJSON := strings.Join(dir.SubPaths, ",")

	_, err := idb.db.Exec(query,
		dir.BlobID,
		dir.DirectoryName,
		dir.DirectoryHash,
		dir.BlockHeight,
		dir.FileCount,
		dir.TotalSize,
		fileTypesJSON,
		subPathsJSON,
	)

	return err
}

func (idb *IndexerDatabase) PutDirectoryFile(directoryID string, fileID string, relativePath string) error {
	query := `
	INSERT OR REPLACE INTO directory_files (directory_id, file_id, relative_path)
	VALUES (?, ?, ?)
	`

	_, err := idb.db.Exec(query, directoryID, fileID, relativePath)
	return err
}

// SearchFiles searches for files by filename with optional filters
func (idb *IndexerDatabase) SearchFiles(query string, filters FileFilters, limit, offset int) ([]*FileIndex, error) {
	sqlQuery := `
	SELECT blob_id, filename, mime_type, file_size, file_hash, block_height, chunk_count, tags
	FROM files 
	WHERE filename LIKE ?
	`
	args := []interface{}{"%" + query + "%"}

	// Add filters
	if filters.MimeType != "" {
		sqlQuery += " AND mime_type = ?"
		args = append(args, filters.MimeType)
	}

	if filters.MinSize > 0 {
		sqlQuery += " AND file_size >= ?"
		args = append(args, filters.MinSize)
	}

	if filters.MaxSize > 0 {
		sqlQuery += " AND file_size <= ?"
		args = append(args, filters.MaxSize)
	}

	sqlQuery += " ORDER BY block_height DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := idb.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileIndex
	for rows.Next() {
		var file FileIndex
		var tagsJSON string

		err := rows.Scan(
			&file.BlobID,
			&file.FileName,
			&file.MimeType,
			&file.FileSize,
			&file.FileHash,
			&file.BlockHeight,
			&file.ChunkCount,
			&tagsJSON,
		)
		if err != nil {
			continue
		}

		// Convert tags back from JSON
		if tagsJSON != "" {
			file.Tags = strings.Split(tagsJSON, ",")
		}

		files = append(files, &file)
	}

	return files, nil
}

func (idb *IndexerDatabase) GetStorageAnalytics() (*StorageAnalytics, error) {
	analytics := &StorageAnalytics{
		MostCommonMimeTypes:  make(map[string]int),
		FileTypeDistribution: make(map[string]uint64),
	}

	// Get block stats
	var avgBlockSizeFloat float64
	err := idb.db.QueryRow(`
		SELECT 
			COUNT(*) as total_blocks,
			COALESCE(SUM(storage_used), 0) as total_storage,
			COALESCE(AVG(storage_used), 0) as avg_block_size
		FROM blocks
	`).Scan(
		&analytics.TotalBlocks,
		&analytics.TotalStorage,
		&avgBlockSizeFloat,
	)
	if err != nil {
		return nil, err
	}

	// Convert average to uint64
	analytics.AvgBlockSize = uint64(avgBlockSizeFloat)

	// Get total chunks
	err = idb.db.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&analytics.TotalChunks)
	if err != nil {
		return nil, err
	}

	// Get actual file count
	err = idb.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&analytics.TotalFiles)
	if err != nil {
		return nil, err
	}

	// Get actual directory count
	err = idb.db.QueryRow("SELECT COUNT(*) FROM directories").Scan(&analytics.TotalDirectories)
	if err != nil {
		return nil, err
	}

	// Get file type distribution
	rows, err := idb.db.Query(`
		SELECT 
			mime_type,
			COUNT(*) as file_count,
			SUM(file_size) as total_size
		FROM files 
		WHERE mime_type IS NOT NULL AND mime_type != ''
		GROUP BY mime_type 
		ORDER BY total_size DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mimeType string
		var fileCount int
		var totalSize uint64

		if err := rows.Scan(&mimeType, &fileCount, &totalSize); err != nil {
			continue
		}

		analytics.MostCommonMimeTypes[mimeType] = fileCount
		analytics.FileTypeDistribution[mimeType] = totalSize
	}

	return analytics, nil
}

func (idb *IndexerDatabase) GetLastIndexedHeight() (uint64, error) {
	var height uint64
	err := idb.db.QueryRow("SELECT COALESCE(MAX(height), 0) FROM blocks").Scan(&height)
	return height, err
}

// GetBlocks returns a paginated list of blocks ordered by height desc
func (idb *IndexerDatabase) GetBlocks(limit, offset int) ([]*BlockIndex, error) {
	query := `
	SELECT height, hash, timestamp, celestia_height, total_chunks, total_files, total_directories, storage_used, parent_hash, dirs_root, files_root, chunks_root, state_root
	FROM blocks 
	ORDER BY height DESC 
	LIMIT ? OFFSET ?
	`

	rows, err := idb.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []*BlockIndex
	for rows.Next() {
		var block BlockIndex
		err := rows.Scan(
			&block.Height,
			&block.Hash,
			&block.Timestamp,
			&block.CelestiaHeight,
			&block.TotalChunks,
			&block.TotalFiles,
			&block.TotalDirectories,
			&block.StorageUsed,
			&block.ParentHash,
			&block.DirsRoot,
			&block.FilesRoot,
			&block.ChunksRoot,
			&block.StateRoot,
		)
		if err != nil {
			continue
		}
		blocks = append(blocks, &block)
	}

	return blocks, nil
}

// GetTotalBlocks returns the total number of blocks
func (idb *IndexerDatabase) GetTotalBlocks() (int, error) {
	var count int
	err := idb.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&count)
	return count, err
}

// GetFiles returns a paginated list of files ordered by block height desc
func (idb *IndexerDatabase) GetFiles(limit, offset int) ([]*FileIndex, error) {
	query := `
	SELECT blob_id, filename, mime_type, file_size, file_hash, block_height, chunk_count, tags
	FROM files 
	ORDER BY block_height DESC 
	LIMIT ? OFFSET ?
	`

	rows, err := idb.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileIndex
	for rows.Next() {
		var file FileIndex
		var tagsJSON string

		err := rows.Scan(
			&file.BlobID,
			&file.FileName,
			&file.MimeType,
			&file.FileSize,
			&file.FileHash,
			&file.BlockHeight,
			&file.ChunkCount,
			&tagsJSON,
		)
		if err != nil {
			continue
		}

		// Convert tags back from JSON
		if tagsJSON != "" {
			file.Tags = strings.Split(tagsJSON, ",")
		}

		files = append(files, &file)
	}

	return files, nil
}

// GetTotalFiles returns the total number of files
func (idb *IndexerDatabase) GetTotalFiles() (int, error) {
	var count int
	err := idb.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&count)
	return count, err
}

// GetDirectories returns a paginated list of directories ordered by timestamp desc
func (idb *IndexerDatabase) GetDirectories(limit, offset int) ([]*DirectoryIndex, error) {
	query := `
	SELECT blob_id, directory_name, directory_hash, block_height, file_count, total_size, file_types, sub_paths
	FROM directories 
	ORDER BY block_height DESC 
	LIMIT ? OFFSET ?
	`

	rows, err := idb.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var directories []*DirectoryIndex
	for rows.Next() {
		var dir DirectoryIndex
		var fileTypesJSON, subPathsJSON string

		err := rows.Scan(
			&dir.BlobID,
			&dir.DirectoryName,
			&dir.DirectoryHash,
			&dir.BlockHeight,
			&dir.FileCount,
			&dir.TotalSize,
			&fileTypesJSON,
			&subPathsJSON,
		)
		if err != nil {
			continue
		}

		// Convert arrays back from JSON
		if fileTypesJSON != "" {
			dir.FileTypes = strings.Split(fileTypesJSON, ",")
		}
		if subPathsJSON != "" {
			dir.SubPaths = strings.Split(subPathsJSON, ",")
		}

		directories = append(directories, &dir)
	}

	return directories, nil
}

// GetTotalDirectories returns the total number of directories
func (idb *IndexerDatabase) GetTotalDirectories() (int, error) {
	var count int
	err := idb.db.QueryRow("SELECT COUNT(*) FROM directories").Scan(&count)
	return count, err
}

// GetDirectoryByID returns a directory by its blob ID
func (idb *IndexerDatabase) GetDirectoryByID(blobID string) (*DirectoryIndex, error) {
	query := `
	SELECT blob_id, directory_name, directory_hash, block_height, file_count, total_size, file_types, sub_paths
	FROM directories 
	WHERE blob_id = ?
	`

	var dir DirectoryIndex
	var fileTypesJSON, subPathsJSON string

	err := idb.db.QueryRow(query, blobID).Scan(
		&dir.BlobID,
		&dir.DirectoryName,
		&dir.DirectoryHash,
		&dir.BlockHeight,
		&dir.FileCount,
		&dir.TotalSize,
		&fileTypesJSON,
		&subPathsJSON,
	)

	if err != nil {
		return nil, err
	}

	// Convert arrays back from JSON
	if fileTypesJSON != "" {
		dir.FileTypes = strings.Split(fileTypesJSON, ",")
	}
	if subPathsJSON != "" {
		dir.SubPaths = strings.Split(subPathsJSON, ",")
	}

	return &dir, nil
}

// GetFilesByBlockHeight returns all files in a specific block
func (idb *IndexerDatabase) GetFilesByBlockHeight(blockHeight uint64) ([]*FileIndex, error) {
	query := `
	SELECT blob_id, filename, mime_type, file_size, file_hash, block_height, chunk_count, tags
	FROM files 
	WHERE block_height = ?
	ORDER BY filename
	`

	rows, err := idb.db.Query(query, blockHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileIndex
	for rows.Next() {
		var file FileIndex
		var tagsJSON string

		err := rows.Scan(
			&file.BlobID,
			&file.FileName,
			&file.MimeType,
			&file.FileSize,
			&file.FileHash,
			&file.BlockHeight,
			&file.ChunkCount,
			&tagsJSON,
		)
		if err != nil {
			continue
		}

		// Convert tags back from JSON
		if tagsJSON != "" {
			file.Tags = strings.Split(tagsJSON, ",")
		}

		files = append(files, &file)
	}

	return files, nil
}

// GetDirectoriesByBlockHeight returns all directories in a specific block
func (idb *IndexerDatabase) GetDirectoriesByBlockHeight(blockHeight uint64) ([]*DirectoryIndex, error) {
	query := `
	SELECT blob_id, directory_name, directory_hash, block_height, file_count, total_size, file_types, sub_paths
	FROM directories 
	WHERE block_height = ?
	ORDER BY directory_name
	`

	rows, err := idb.db.Query(query, blockHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var directories []*DirectoryIndex
	for rows.Next() {
		var dir DirectoryIndex
		var fileTypesJSON, subPathsJSON string

		err := rows.Scan(
			&dir.BlobID,
			&dir.DirectoryName,
			&dir.DirectoryHash,
			&dir.BlockHeight,
			&dir.FileCount,
			&dir.TotalSize,
			&fileTypesJSON,
			&subPathsJSON,
		)
		if err != nil {
			continue
		}

		// Convert arrays back from JSON
		if fileTypesJSON != "" {
			dir.FileTypes = strings.Split(fileTypesJSON, ",")
		}
		if subPathsJSON != "" {
			dir.SubPaths = strings.Split(subPathsJSON, ",")
		}

		directories = append(directories, &dir)
	}

	return directories, nil
}

// PutChunkIndex stores a chunk index
func (idb *IndexerDatabase) PutChunkIndex(chunk *ChunkIndex) error {
	query := `
	INSERT OR REPLACE INTO chunks 
	(blob_id, block_height, chunk_index, chunk_size, chunk_hash)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err := idb.db.Exec(query,
		chunk.BlobID,
		chunk.BlockHeight,
		chunk.Index,
		chunk.ChunkSize,
		chunk.ChunkHash,
	)

	return err
}

// GetChunksByBlockHeight returns all chunks in a specific block
func (idb *IndexerDatabase) GetChunksByBlockHeight(blockHeight uint64) ([]*ChunkIndex, error) {
	query := `
	SELECT blob_id, block_height, chunk_index, chunk_size, chunk_hash
	FROM chunks 
	WHERE block_height = ?
	ORDER BY chunk_index
	`

	rows, err := idb.db.Query(query, blockHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []*ChunkIndex
	for rows.Next() {
		var chunk ChunkIndex
		err := rows.Scan(
			&chunk.BlobID,
			&chunk.BlockHeight,
			&chunk.Index,
			&chunk.ChunkSize,
			&chunk.ChunkHash,
		)
		if err != nil {
			continue
		}
		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}
