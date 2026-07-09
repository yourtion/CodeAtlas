package fixtures

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// fakeSymbolRepo 是 symbolNameLookup 的内存假实现，按 name 返回预设候选。
// 不依赖 DB，故 lookupSymbolInFile 的消歧逻辑可纯单测。
type fakeSymbolRepo struct {
	byName map[string][]*models.Symbol
	// calls 记录 GetByExactName 被调用的 name，便于断言查询次数。
	calls []string
	// err 若非 nil，GetByExactName 直接返回（验证错误传播）。
	err error
}

func (f *fakeSymbolRepo) GetByExactName(ctx context.Context, name string) ([]*models.Symbol, error) {
	f.calls = append(f.calls, name)
	if f.err != nil {
		return nil, f.err
	}
	return f.byName[name], nil
}

// newFileOfFunc 构造一个 file_id -> path 的闭包，并统计调用次数（验证缓存命中）。
func newFileOfFunc(paths map[string]string) (func(ctx context.Context, fileID string) (string, error), *int) {
	calls := 0
	return func(ctx context.Context, fileID string) (string, error) {
		calls++
		return paths[fileID], nil
	}, &calls
}

// TestLookupSymbolInFile_NoCandidates 候选为空时返回空 ID。
func TestLookupSymbolInFile_NoCandidates(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{}}
	fileOf, _ := newFileOfFunc(map[string]string{})

	sid, fileID, err := lookupSymbolInFile(ctx, repo, fileOf, "missing", []string{"a.go"})
	require.NoError(t, err)
	assert.Empty(t, sid)
	assert.Empty(t, fileID)
}

// TestLookupSymbolInFile_SingleCandidate 单候选 + 偏好路径命中 -> 直接取该候选。
// 实现对单候选无快路径：只要 fileOf != nil 且 preferPaths 非空，仍会逐候选解析 path。
func TestLookupSymbolInFile_SingleCandidate(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-1", FileID: "file-1", Name: "foo"},
		},
	}}
	fileOf, callsPtr := newFileOfFunc(map[string]string{"file-1": "src/a.go"})

	sid, fileID, err := lookupSymbolInFile(ctx, repo, fileOf, "foo", []string{"src/a.go"})
	require.NoError(t, err)
	assert.Equal(t, "sym-1", sid)
	assert.Equal(t, "file-1", fileID)
	// 单候选仍会解析一次 file path（无快路径）。
	assert.Equal(t, 1, *callsPtr)
}

// TestLookupSymbolInFile_MultiSameFile 多候选且偏好路径命中 -> 取命中的、symbol_id 最小者。
func TestLookupSymbolInFile_MultiSameFile(t *testing.T) {
	ctx := context.Background()
	// 两个候选：一个在偏好路径 src/Repo.kt，一个在 src/Other.kt。
	// 同文件内还有两个（模拟 Kotlin/Java 同名方法各落到自己文件）。
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"findById": {
			{SymbolID: "sym-kt", FileID: "file-kt", Name: "findById"},
			{SymbolID: "sym-java", FileID: "file-java", Name: "findById"},
		},
	}}
	fileOf, _ := newFileOfFunc(map[string]string{
		"file-kt":   "tests/fixtures/kotlin/UserRepository.kt",
		"file-java": "tests/fixtures/java/UserRepository.java",
	})

	// 偏好 Kotlin 文件 -> 应取 sym-kt。
	sid, fileID, err := lookupSymbolInFile(ctx, repo, fileOf, "findById",
		[]string{"tests/fixtures/kotlin/UserRepository.kt"})
	require.NoError(t, err)
	assert.Equal(t, "sym-kt", sid)
	assert.Equal(t, "file-kt", fileID)

	// 偏好 Java 文件 -> 应取 sym-java（同 fileOf 复用，无额外状态）。
	sid, fileID, err = lookupSymbolInFile(ctx, repo, fileOf, "findById",
		[]string{"tests/fixtures/java/UserRepository.java"})
	require.NoError(t, err)
	assert.Equal(t, "sym-java", sid)
	assert.Equal(t, "file-java", fileID)
}

// TestLookupSymbolInFile_MultiSameFilePicksSmallestSymbolID 偏好路径命中多个候选时，
// 按 symbol_id 升序取首个（确定性）。
func TestLookupSymbolInFile_MultiSameFilePicksSmallestSymbolID(t *testing.T) {
	ctx := context.Background()
	// 两个候选都在同一偏好路径下；sym-b 字典序大于 sym-a，应取 sym-a。
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-b", FileID: "file-1", Name: "foo"},
			{SymbolID: "sym-a", FileID: "file-1", Name: "foo"},
		},
	}}
	fileOf, _ := newFileOfFunc(map[string]string{"file-1": "src/a.go"})

	sid, fileID, err := lookupSymbolInFile(ctx, repo, fileOf, "foo", []string{"src/a.go"})
	require.NoError(t, err)
	assert.Equal(t, "sym-a", sid)
	assert.Equal(t, "file-1", fileID)
}

// TestLookupSymbolInFile_MultiNoPreferredMatch 偏好路径无命中 -> 退化为首个（symbol_id 升序）。
func TestLookupSymbolInFile_MultiNoPreferredMatch(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-z", FileID: "file-1", Name: "foo"},
			{SymbolID: "sym-y", FileID: "file-2", Name: "foo"},
		},
	}}
	fileOf, _ := newFileOfFunc(map[string]string{
		"file-1": "src/a.go",
		"file-2": "src/b.go",
	})

	// 偏好一个不存在的路径 -> 无命中 -> 取 symbol_id 升序首个 sym-y。
	sid, _, err := lookupSymbolInFile(ctx, repo, fileOf, "foo", []string{"src/none.go"})
	require.NoError(t, err)
	assert.Equal(t, "sym-y", sid)
}

// TestLookupSymbolInFile_FileOfNil 退化分支：fileOf 为 nil 时不 panic，直接取首个候选。
// 对应 ResolveTruthIDs 传 nil fileRepo 的旧行为。
func TestLookupSymbolInFile_FileOfNil(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-z", FileID: "file-1", Name: "foo"},
			{SymbolID: "sym-y", FileID: "file-2", Name: "foo"},
		},
	}}

	// fileOf=nil + 非空 preferPaths -> 走退化分支，取 symbol_id 升序首个 sym-y。
	sid, fileID, err := lookupSymbolInFile(ctx, repo, nil, "foo", []string{"src/a.go"})
	require.NoError(t, err)
	assert.Equal(t, "sym-y", sid)
	assert.Equal(t, "file-2", fileID)
}

// TestLookupSymbolInFile_EmptyPreferPaths preferPaths 为空 -> 退化为首个候选。
func TestLookupSymbolInFile_EmptyPreferPaths(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-z", FileID: "file-1", Name: "foo"},
			{SymbolID: "sym-y", FileID: "file-2", Name: "foo"},
		},
	}}
	fileOf, callsPtr := newFileOfFunc(map[string]string{"file-1": "src/a.go"})

	// preferPaths 空 -> 不解析 file path，直接取首个。
	sid, _, err := lookupSymbolInFile(ctx, repo, fileOf, "foo", nil)
	require.NoError(t, err)
	assert.Equal(t, "sym-y", sid)
	assert.Equal(t, 0, *callsPtr, "preferPaths 空时不应调用 fileOf")
}

// TestLookupSymbolInFile_PreferPathsFiltersEmptyString preferPaths 里的空串被忽略
// （对应 edge.SourceFilePath 未回填的情形，不应误匹配 file_id 解析为空的候选）。
func TestLookupSymbolInFile_PreferPathsFiltersEmptyString(t *testing.T) {
	ctx := context.Background()
	// 两个候选：file-1 的 path 为空（fileOf 返回 ""），file-2 的 path 为 src/b.go。
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-1", FileID: "file-1", Name: "foo"},
			{SymbolID: "sym-2", FileID: "file-2", Name: "foo"},
		},
	}}
	fileOf, _ := newFileOfFunc(map[string]string{
		"file-1": "",
		"file-2": "src/b.go",
	})

	// preferPaths 含空串（模拟 SourceFilePath 未回填）+ FixtureFile。
	// 实现用 `if p != ""` 过滤空串，故 prefer 集合仅含 "src/b.go"。
	// file-1 的 path 为 "" 不在集合里 -> 不命中；file-2 path="src/b.go" 命中 -> 取 sym-2。
	sid, fileID, err := lookupSymbolInFile(ctx, repo, fileOf, "foo",
		[]string{"", "src/b.go"})
	require.NoError(t, err)
	assert.Equal(t, "sym-2", sid)
	assert.Equal(t, "file-2", fileID)
}

// TestLookupSymbolInFile_FileOfError fileOf 返回错误时向上传播。
func TestLookupSymbolInFile_FileOfError(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-1", FileID: "file-1", Name: "foo"},
		},
	}}
	sentinel := errors.New("db down")
	fileOf := func(ctx context.Context, fileID string) (string, error) {
		return "", sentinel
	}

	// 多候选 + 非空 preferPaths 才会调用 fileOf；构造两个候选触发解析。
	repo.byName["foo"] = append(repo.byName["foo"], &models.Symbol{SymbolID: "sym-2", FileID: "file-2", Name: "foo"})
	_, _, err := lookupSymbolInFile(ctx, repo, fileOf, "foo", []string{"src/a.go"})
	require.ErrorIs(t, err, sentinel)
}

// TestLookupSymbolInFile_RepoError 仓库查询出错时向上传播。
func TestLookupSymbolInFile_RepoError(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("repo down")
	repo := &fakeSymbolRepo{err: sentinel}
	fileOf, _ := newFileOfFunc(map[string]string{})

	_, _, err := lookupSymbolInFile(ctx, repo, fileOf, "foo", []string{"src/a.go"})
	require.ErrorIs(t, err, sentinel)
}

// TestLookupSymbolInFile_FileOfCalledOncePerCandidate 验证 lookupSymbolInFile
// 内逐候选调用 fileOf（而非逐候选直接打 DB）。配合 ResolveTruthIDs 的缓存闭包，
// 同一 file_id 的重复解析会命中缓存——此测试用计数器验证「每候选各调一次」。
func TestLookupSymbolInFile_FileOfCalledOncePerCandidate(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSymbolRepo{byName: map[string][]*models.Symbol{
		"foo": {
			{SymbolID: "sym-1", FileID: "file-1", Name: "foo"},
			{SymbolID: "sym-2", FileID: "file-1", Name: "foo"}, // 同 file-1
			{SymbolID: "sym-3", FileID: "file-2", Name: "foo"},
		},
	}}
	fileOf, callsPtr := newFileOfFunc(map[string]string{
		"file-1": "src/a.go",
		"file-2": "src/b.go",
	})

	_, _, err := lookupSymbolInFile(ctx, repo, fileOf, "foo", []string{"src/a.go"})
	require.NoError(t, err)
	// 3 个候选各解析一次 file_id -> path（lookupSymbolInFile 自身不缓存，
	// 缓存由调用方 ResolveTruthIDs 的 pathOf 闭包提供）。
	assert.Equal(t, 3, *callsPtr, "应逐候选调用 fileOf 一次")
}

// TestResolveTruthIDs_FilesCacheSharedAcrossEdges 集成性单测：用 fakeSymbolRepo +
// 内存 fileOf 验证 ResolveTruthIDs 的多层级消歧与缓存共享。
// 不需要 DB——fileRepo 参数传 nil（走 fileOf 退化路径之外，symbolRepo 用 fake）。
//
// 注意：ResolveTruthIDs 当前签名接受 *models.SymbolRepository / *models.FileRepository
// 具体类型，故无法直接注入 fakeSymbolRepo。此测试改为验证 lookupSymbolInFile 的
// 端到端语义（source 同文件 / target 同 source / 无命中退化）已在上述子测试覆盖；
// ResolveTruthIDs 的 DB 集成验证由 tests/integration/quality_gate_test.go 承担。
