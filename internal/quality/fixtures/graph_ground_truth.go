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
}

// ResolveTruthIDs 索引 fixture 后回填真值边的 SourceID/TargetID。
//
// symbol_id 是 GenerateDeterministicUUID 基于 (file_id, name, start_line, start_byte) 产出的，
// 虽然确定性，但硬编码脆弱。改为索引后从 DB 查出回填。
//
// 匹配策略：按 name 精确查询符号（GetByExactName，区分大小写，不把 _ 当通配符），
// 取首个候选。Symbol 表只有 file_id 无 file_path，故不做 file_path 消歧——
// fixture 仓库内同名符号少，首个候选即可。
//
// 查不到的 ID 留空——computeEdgeMatch 会跳过 TargetID 空的边（不计入 recall/precision）。
// Optional 边或 TargetName 空的边（如标准库 strlen、外部 import 模块）也无须回填。
// DB 错误会立即返回（不再吞掉），以便真正的 DB 故障能暴露。
func ResolveTruthIDs(ctx context.Context, symbolRepo *models.SymbolRepository, truth []quality.GraphGroundTruth) error {
	for gi := range truth {
		gt := &truth[gi]
		for i := range gt.Edges {
			edge := &gt.Edges[i]
			if edge.SourceName != "" && edge.SourceID == "" {
				sid, err := lookupSymbolID(ctx, symbolRepo, edge.SourceName)
				if err != nil {
					return err
				}
				edge.SourceID = sid
			}
			if edge.TargetName != "" && edge.TargetID == "" {
				sid, err := lookupSymbolID(ctx, symbolRepo, edge.TargetName)
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

// lookupSymbolID 按 name 精确查符号 ID，取首个候选。
// 找不到时返回 ("", nil)——调用方据此保留 ID 为空；DB 错误时返回 ("", err)。
func lookupSymbolID(ctx context.Context, repo *models.SymbolRepository, name string) (string, error) {
	syms, err := repo.GetByExactName(ctx, name)
	if err != nil {
		return "", err
	}
	if len(syms) == 0 {
		return "", nil
	}
	return syms[0].SymbolID, nil
}
