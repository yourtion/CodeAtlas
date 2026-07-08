// Package fixtures 存放评估真值（ground truth）。
//
// 真值来源：从 tests/integration/call_analysis_fixtures_test.go 里散落的
// expectedXxxCalls/expectedXxxImports/expectedXxxFrameworks 列表系统化迁移而来，
// 后经集成测试 TestQualityGate_FixtureMode 在真 DB 上校准，与索引器实际入库的边对齐。
//
// 匹配按 (SourceID, EdgeType, TargetID) 三元组进行 symbol_id 精确匹配，由
// ResolveTruthIDs 在索引 fixture 后从 DB 回填 SourceID/TargetID（解决 C++ 重载同名问题）。
// SourceName/TargetName 仅保留用于调试日志与符号查找；target 悬空时 TargetID 为空，
// TargetName 回退到 edges.target_module（对 import 边即模块名）。
//
// 跨文件符号消解（已修复）：
//
//	internal/schema.SchemaMapper 采用 CollectSymbols + ResolveEdges 两遍扫描——
//	第一遍累积全仓库符号候选与 import 关系，第二遍用候选集解析所有边。
//	故「跨文件」调用边（如 cpp_calls_c.cpp 的 processData -> c_library.h 的
//	c_process_string）的 target_id 现在能正确消解到被调用方符号。
//	同一文件内 target 为标准库（strlen/malloc 等，源文件无定义）的边则保留为悬空
//	（target_id 空），target_name 回退到 target_module（多数亦为空）。
//
//	Optional=true 的边不计入 edge_recall 漏检（如标准库/外部运行时函数），
//
// 但会出现在真值集合里，使 edge_precision 不被这些「合法但无法消解」的边拉低。
//
// Chains 是端到端调用链，用于 call_chain_connectivity 指标。跨文件链（如
// main -> c_process_string，经 cpp_calls_c.cpp 的 processData 跨文件到达 c_library.h）
// 现在可达，可标注跨文件链路。
package fixtures

import (
	"context"
	"sort"

	"github.com/yourtionguo/CodeAtlas/internal/quality"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// CallAnalysisGroundTruth 是 call_analysis fixture 集的依赖图真值。
//
// 下列真值由集成测试 indexRealFixtures 索引 tests/fixtures/ 下文件后，
// 通过 models.ListExtractedEdges 实测校准得到（见 graph_evaluator_test.go 旁的
// quality_gate_test.go）。新增 fixture 时务必先在真 DB 上核对实际入库的符号名/边，
// 再据此增补——符号名以解析器实际产出为准（cpp 方法/构造器不带类前缀，
// 析构器带 ~，objc 用选择器名等）。
var CallAnalysisGroundTruth = []quality.GraphGroundTruth{
	// ──────────────────────────────────────────────────────────────
	// 1. cpp_calls_c.cpp —— C++ 包装类调用 C 库函数（跨文件到 c_library.h）。
	//    main 调用 wrapper 实例方法（同文件）；CWrapper 的构造/析构、processData、
	//    calculate、useStruct、processCData 直接调用 c_library.h 里的 C 函数。
	//    标准库 strlen/malloc/strcpy/printf 等保留为悬空（Optional）。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/cpp/cpp_calls_c.cpp",
		Edges: []quality.ExpectedEdge{
			// main -> wrapper 实例方法（同文件内）。
			{SourceName: "main", EdgeType: "call", TargetName: "processData"},
			{SourceName: "main", EdgeType: "call", TargetName: "calculate"},
			{SourceName: "main", EdgeType: "call", TargetName: "useStruct"},

			// 跨文件调用到 c_library.h 的 C 函数。
			{SourceName: "CWrapper", EdgeType: "call", TargetName: "c_init"},     // 构造器
			{SourceName: "~CWrapper", EdgeType: "call", TargetName: "c_free"},    // 析构器
			{SourceName: "~CWrapper", EdgeType: "call", TargetName: "c_cleanup"}, // 析构器
			{SourceName: "processData", EdgeType: "call", TargetName: "c_process_string"},
			{SourceName: "calculate", EdgeType: "call", TargetName: "c_add"},
			{SourceName: "calculate", EdgeType: "call", TargetName: "c_multiply"},
			{SourceName: "useStruct", EdgeType: "call", TargetName: "c_init_struct"},
			{SourceName: "useStruct", EdgeType: "call", TargetName: "c_process_struct"},
			{SourceName: "useStruct", EdgeType: "call", TargetName: "c_free_struct"},
			{SourceName: "processCData", EdgeType: "call", TargetName: "c_log_message"},
			{SourceName: "processCData", EdgeType: "call", TargetName: "c_validate_input"},

			// 标准库函数（悬空：源文件无定义，target_id 为空）。
			// 提到不算漏（Optional），计入真值集合以免拉低 precision。
			{SourceName: "main", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "processData", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "processCData", EdgeType: "call", TargetName: "", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			// main -> processData：同一文件内、入库 call 边可达。
			{StartName: "main", EndName: "processData",
				StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/cpp/cpp_calls_c.cpp"},
			// main -> useStruct：同一文件内可达。
			{StartName: "main", EndName: "useStruct",
				StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/cpp/cpp_calls_c.cpp"},
			// 跨文件链：main -> processData（同文件）-> c_process_string（跨文件到 c_library.h）。
			{StartName: "main", EndName: "c_process_string",
				StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/cpp/c_library.h"},
			// 跨文件链：main -> calculate -> c_add（c_library.h）。
			{StartName: "main", EndName: "c_add",
				StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/cpp/c_library.h"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 2. kotlin_calls_java.kt —— import 边（source 为包名 com.example.interop，
	//    target 为 java 标准库类，外部依赖未索引，target_id 悬空，target_name 回退到
	//    target_module）。这些 import 边 source 非空故可入库，Optional 让其不计漏。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/kotlin/kotlin_calls_java.kt",
		Edges: []quality.ExpectedEdge{
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "java.util.ArrayList", Optional: true},
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "java.util.HashMap", Optional: true},
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "java.util.Date", Optional: true},
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "java.text.SimpleDateFormat", Optional: true},
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "java.io.File", Optional: true},
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "java.io.FileReader", Optional: true},
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "java.io.BufferedReader", Optional: true},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 3. typescript_calls_js.ts —— import 边（source 为模块名 typescript_calls_js，
	//    target 为 JS 模块路径/外部包，target_id 悬空，target_name 回退到 target_module）。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/js/typescript_calls_js.ts",
		Edges: []quality.ExpectedEdge{
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "./legacy-module.js", Optional: true},
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "./utils.js", Optional: true},
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "./default-export.js", Optional: true},
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "fs", Optional: true},
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "path", Optional: true},
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "util", Optional: true},
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "old-js-library", Optional: true},
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "https://api.example.com/data", Optional: true},

			// JS 运行时 API 调用（悬空：console/setTimeout/Promise/fetch 等）。
			{SourceName: "useJavaScriptGlobals", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptArrays", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptObjects", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptStrings", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptDate", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptJSON", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptMath", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptLocalStorage", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptRegExp", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptFetch", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "useJavaScriptRequire", EdgeType: "call", TargetName: "", Optional: true},
			{SourceName: "callJavaScriptDynamic", EdgeType: "call", TargetName: "", Optional: true},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 4. simple_c_calls.m —— ObjC 类引用自身（SimpleWrapper 的 @interface/@implementation
	//    互引，target 同文件内可消解）。C 函数调用（c_add/c_log/strlen/printf）解析器
	//    当前未提取为 call 边，故不标注。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/objc/simple_c_calls.m",
		Edges: []quality.ExpectedEdge{
			{SourceName: "SimpleWrapper", EdgeType: "reference", TargetName: "SimpleWrapper"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 5. swift_calls_objc.swift —— 继承 UIKit/Foundation 基类（外部框架，target 悬空）
	//    与类自引用。继承/引用边 source 为类名，target 为外部基类（未索引）。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/swift/swift_calls_objc.swift",
		Edges: []quality.ExpectedEdge{
			{SourceName: "SwiftViewController", EdgeType: "extends", TargetName: "", Optional: true},
			{SourceName: "SwiftViewController", EdgeType: "reference", TargetName: "", Optional: true},
			{SourceName: "BridgedClass", EdgeType: "extends", TargetName: "", Optional: true},
			{SourceName: "BridgedClass", EdgeType: "reference", TargetName: "", Optional: true},
			{SourceName: "useCoreFoundation", EdgeType: "call", TargetName: "", Optional: true},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 6. UserService.kt —— Kotlin 跨文件方法调用。
	//    UserService 的方法调用 UserRepository 的方法（跨文件）。
	//    方法符号展平后，这些调用边的 target_id 应能消解到 UserRepository 的方法符号。
	//    Optional 的边为 Kotlin 标准库调用（toList/find/removeIf 等，源文件无定义）。
	//
	// ResolveTruthIDs 现按各条目的 FixtureFile 做 source/target 同文件消歧，
	// 故 Kotlin/Java 同名方法可分别落到各自文件的符号上。
	//
	// 注：这里不挂 Chains——getAllUsers→findAll 的 target 在索引器消歧时因
	// Kotlin/Java 两个 UserRepository 同名（import 模块基名撞）会落到 Java 的 findAll，
	// 使「getAllUsers(KT) → findAll(KT repo)」链不可达。该跨语言消歧弱点超出本任务
	// （方法符号展平）范围；跨文件方法调用链的连通性由第 8 条（Java findById→getId）
	// 验证，后者终点 getId 仅存在于 User.java，无同名歧义。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
		Edges: []quality.ExpectedEdge{
			{SourceName: "getAllUsers", EdgeType: "call", TargetName: "findAll"},
			{SourceName: "getUserById", EdgeType: "call", TargetName: "findById"},
			{SourceName: "createUser", EdgeType: "call", TargetName: "save"},
			{SourceName: "deleteUser", EdgeType: "call", TargetName: "delete"},
			// Kotlin 标准库调用（悬空）
			{SourceName: "getAllUsers", EdgeType: "call", TargetName: "", Optional: true},
		},
	},
	// ──────────────────────────────────────────────────────────────
	// 7. UserRepository.kt —— Kotlin 仓库方法间调用。
	//    findById/delete 内部用 it.id（Kotlin 属性访问，消解到 User.kt 的 id 字段）；
	//    save 调用 users.add(user)（同文件内 Kotlin 标准库 List.add）。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/repository/UserRepository.kt",
		Edges: []quality.ExpectedEdge{
			// findById/delete 内部 it.id 属性访问（消解到 User.kt 的 id 字段）
			{SourceName: "findById", EdgeType: "call", TargetName: "id"},
			{SourceName: "delete", EdgeType: "call", TargetName: "id"},
			// save 调用 users.add(user)（Kotlin 标准库 List.add）
			{SourceName: "save", EdgeType: "call", TargetName: "add"},
			// Kotlin 标准库调用（悬空）
			{SourceName: "findAll", EdgeType: "call", TargetName: "", Optional: true},
		},
	},
	// ──────────────────────────────────────────────────────────────
	// 8. UserRepository.java —— Java 仓库方法间调用（跨文件到 User.java）。
	//    findById/delete 内部调用 user.getId()（Java getter，跨文件到 User.java）；
	//    save 调用 users.add(user)（Java 标准库 List.add）。
	//    方法符号展平后可验证 Java 方法级符号入库与跨文件方法调用消解。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java",
		Edges: []quality.ExpectedEdge{
			// findById/delete 内部调用 user.getId()（跨文件到 User.java）
			{SourceName: "findById", EdgeType: "call", TargetName: "getId"},
			{SourceName: "delete", EdgeType: "call", TargetName: "getId"},
			// save 调用 users.add(user)（Java 标准库 List.add）
			{SourceName: "save", EdgeType: "call", TargetName: "add"},
			// Java 标准库调用（悬空）
			{SourceName: "findAll", EdgeType: "call", TargetName: "", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			// Java 跨文件方法调用链：findById 调用 user.getId()（跨文件到 User.java）。
			{StartName: "findById", EndName: "getId",
				StartFile: "tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java",
				EndFile:   "tests/fixtures/java/src/main/java/com/example/myapp/model/User.java"},
		},
	},
}

// ResolveTruthIDs 索引 fixture 后回填真值边的 SourceID/TargetID。
//
// symbol_id 是 GenerateDeterministicUUID 基于 (file_id, name, start_line, start_byte) 产出的，
// 虽然确定性，但硬编码脆弱。改为索引后从 DB 查出回填。
//
// 匹配策略：按 name 精确查询符号（GetByExactName，区分大小写，不把 _ 当通配符）。
// 单候选直接取；多候选时近似「同文件优先」消歧（见 lookupSymbolInFile 文档）——
//   - target 优先取与「source 符号所在文件」同文件的候选（如 findById→id，
//     二者同在 UserRepository.kt），其次取与 FixtureFile 同文件的候选；
//   - 仍歧义则按 symbol_id 升序取首个（确定性）。
//
// 注意：此消歧仅近似 SchemaMapper.disambiguate 的「同文件优先」第一层，并不包含
// disambiguate 的 import-path 匹配层（truth 解析阶段拿不到源文件的 import 表，故无法
// 复刻索引器的 import-match 消歧）。对索引器最终经 import-match 解析的边，真值可能
// 落到不同符号上而偏离——这些差异会在 edge_recall/precision 里体现，属已知偏差。
// fileRepo 用于把 file_id 解析为 path 做比较；传 nil 则退化为首个候选（旧行为）。
//
// 查不到的 ID 留空——computeEdgeMatch 会跳过 TargetID 空的边（不计入 recall/precision）。
// Optional 边或 TargetName 空的边（如标准库 strlen、外部 import 模块）也无须回填。
// DB 错误会立即返回（不再吞掉），以便真正的 DB 故障能暴露。
func ResolveTruthIDs(ctx context.Context, symbolRepo *models.SymbolRepository, fileRepo *models.FileRepository, truth []quality.GraphGroundTruth) error {
	// file_id -> path 缓存（避免重复查询）。供 source 路径回填与候选消歧共用——
	// lookupSymbolInFile 内逐候选解析 file_id -> path 也走此缓存（通过 fileOf 闭包）。
	pathCache := make(map[string]string)
	pathOf := func(ctx context.Context, fileID string) (string, error) {
		if p, ok := pathCache[fileID]; ok {
			return p, nil
		}
		if fileRepo == nil {
			return "", nil
		}
		f, err := fileRepo.GetByID(ctx, fileID)
		if err != nil {
			return "", err
		}
		var p string
		if f != nil {
			p = f.Path
		}
		pathCache[fileID] = p
		return p, nil
	}

	for gi := range truth {
		gt := &truth[gi]
		for i := range gt.Edges {
			edge := &gt.Edges[i]
			if edge.SourceName != "" && edge.SourceID == "" {
				// source 优先取与 FixtureFile 同文件的候选——各真值条目的
				// FixtureFile 标明了边所在的源文件，据此把同名符号（如 Kotlin 与
				// Java 各自的 findById）区分开。
				sid, srcFileID, err := lookupSymbolInFile(ctx, symbolRepo, pathOf, edge.SourceName, []string{gt.FixtureFile})
				if err != nil {
					return err
				}
				edge.SourceID = sid
				// 记下 source 所在文件路径，供 target 同文件消歧。
				if srcFileID != "" {
					p, err := pathOf(ctx, srcFileID)
					if err != nil {
						return err
					}
					edge.SourceFilePath = p
				}
			}
			if edge.TargetName != "" && edge.TargetID == "" {
				// target 优先与 source 同文件，其次与 FixtureFile 同文件。
				prefer := []string{edge.SourceFilePath, gt.FixtureFile}
				sid, _, err := lookupSymbolInFile(ctx, symbolRepo, pathOf, edge.TargetName, prefer)
				if err != nil {
					return err
				}
				edge.TargetID = sid
			}
		}
		// Chains 用 name+file 查询连通性，不需回填 ID
	}
	return nil
}

// symbolNameLookup 抽象「按精确名字查符号」的仓库操作，便于 lookupSymbolInFile
// 在不依赖 *models.SymbolRepository 具体类型的情况下被单测（用内存假实现替换）。
// *models.SymbolRepository 天然满足此接口。
type symbolNameLookup interface {
	GetByExactName(ctx context.Context, name string) ([]*models.Symbol, error)
}

// lookupSymbolInFile 在 name 命中的候选里，优先取 file_id 对应 path 命中 preferPaths
// 任一者的候选；无命中（或 fileOf 为 nil / preferPaths 为空）则取首个（按 symbol_id 升序）。
// fileOf 用于解析候选 file_id -> path；为 nil 时退化为首个候选。返回 (symbolID, fileID)。
//
// preferPaths 是一个无序集合（非优先级列表）——多条目间无先后优先级，仅用于判定
// 「候选所在文件是否属于偏好集合」。
//
// 与 SchemaMapper.disambiguate 的关系：此函数近似 disambiguate 的「同文件优先」第一层
// （preferPaths 即真值侧的「source 同文件 / FixtureFile」集合），但 **不包含**
// disambiguate 的第二层 import-path 匹配——truth 解析阶段拿不到源文件的 import 表，
// 无法复刻索引器的 import-match 消歧。故对索引器最终经 import-match 解析的边，真值
// 可能落到不同符号上而偏离（已知偏差）。fileOf 解析候选 file_id -> path；调用方
// （ResolveTruthIDs）在其实现里带 file_id -> path 缓存，故 source 与候选的路径解析
// 共用同一缓存，避免逐候选重复查 DB。
func lookupSymbolInFile(ctx context.Context, repo symbolNameLookup, fileOf func(ctx context.Context, fileID string) (string, error), name string, preferPaths []string) (string, string, error) {
	syms, err := repo.GetByExactName(ctx, name)
	if err != nil {
		return "", "", err
	}
	if len(syms) == 0 {
		return "", "", nil
	}
	if fileOf != nil && len(preferPaths) > 0 {
		prefer := make(map[string]bool, len(preferPaths))
		for _, p := range preferPaths {
			if p != "" {
				prefer[p] = true
			}
		}
		var matched []*models.Symbol
		for _, s := range syms {
			p, err := fileOf(ctx, s.FileID)
			if err != nil {
				return "", "", err
			}
			if prefer[p] {
				matched = append(matched, s)
			}
		}
		if len(matched) > 0 {
			sortSymbolsByID(matched)
			return matched[0].SymbolID, matched[0].FileID, nil
		}
	}
	sortSymbolsByID(syms)
	return syms[0].SymbolID, syms[0].FileID, nil
}

// sortSymbolsByID 按 SymbolID 升序排序（确定性消歧）。
func sortSymbolsByID(syms []*models.Symbol) {
	sort.Slice(syms, func(i, j int) bool { return syms[i].SymbolID < syms[j].SymbolID })
}
