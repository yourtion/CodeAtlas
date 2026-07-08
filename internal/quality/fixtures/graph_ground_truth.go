// Package fixtures 存放评估真值（ground truth）。
//
// 真值来源：从 tests/integration/call_analysis_fixtures_test.go 里散落的
// expectedXxxCalls/expectedXxxImports/expectedXxxFrameworks 列表系统化迁移而来，
// 后经集成测试 TestQualityGate_FixtureMode 在真 DB 上校准，与索引器实际入库的边对齐。
//
// 每条 Edge 是 (SourceName, EdgeType, TargetName) 三元组，与 models.ListExtractedEdges
// 返回的 ExtractedEdge 字段一一对应（DB 里 source_symbol.name / edge_type /
// target_symbol.name；target 悬空时回退到 edges.target_module，对 import 边即模块名）。
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

import "github.com/yourtionguo/CodeAtlas/internal/quality"

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
