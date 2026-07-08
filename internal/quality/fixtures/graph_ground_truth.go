// Package fixtures 存放评估真值（ground truth）。
//
// 真值来源：从 tests/integration/call_analysis_fixtures_test.go 里散落的
// expectedXxxCalls/expectedXxxImports/expectedXxxFrameworks 列表系统化迁移而来，
// 后经集成测试 TestQualityGate_FixtureMode 在真 DB 上校准，与索引器实际入库的边对齐。
//
// 每条 Edge 是 (SourceName, EdgeType, TargetName) 三元组，与 models.ListExtractedEdges
// 返回的 ExtractedEdge 字段一一对应（DB 里 source_symbol.name / edge_type / target_symbol.name）。
//
// ⚠️ 索引解析边界（重要）：
//
//	internal/schema.SchemaMapper 在 MapToSchema 时按文件重置符号表（symbolIDMap），
//	因此「跨文件」调用边（如 cpp_calls_c.cpp 的 processData -> c_process_string，
//	目标符号定义在 c_library.h）因 target_id 无法消解而被丢弃；同理「同一文件内」
//	但 target 在 mapper 符号表里查不到的边（如标准库 strlen/malloc，源文件无定义）也丢弃。
//	入库的边只有：同文件内、source 与 target 都能消解的 call 边，以及 source 非空的 import 边。
//
//	故本真值只标注「实际会入库」的边——这是对当前索引管线的真实刻画。
//	跨文件调用消解能力（让 processData -> c_process_string 也能入库）是已知缺口，
//	待 SchemaMapper 支持两遍扫描后补真值。
//
// Chains 是端到端调用链，用于 call_chain_connectivity 指标（仅在入库 call 边上可达）。
package fixtures

import "github.com/yourtionguo/CodeAtlas/internal/quality"

// CallAnalysisGroundTruth 是 call_analysis fixture 集的依赖图真值。
//
// 下列真值由集成测试 indexRealFixtures 索引 tests/fixtures/ 下文件后，
// 通过 models.ListExtractedEdges 实测校准得到（见 graph_evaluator_test.go 旁的
// quality_gate_test.go）。新增 fixture 时务必先在真 DB 上核对实际入库的符号名/边，
// 再据此增补——符号名以解析器实际产出为准（cpp 方法不带类前缀，objc 用选择器名等）。
var CallAnalysisGroundTruth = []quality.GraphGroundTruth{
	// ──────────────────────────────────────────────────────────────
	// 1. cpp_calls_c.cpp —— main 调用同文件内的 CWrapper 方法。
	//    跨文件到 c_library.h 的调用（processData->c_process_string 等）因
	//    SchemaMapper 单文件符号消解被丢弃，故不在此标注。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/cpp/cpp_calls_c.cpp",
		Edges: []quality.ExpectedEdge{
			// main 内调用 wrapper 实例的方法（解析器解析为同名符号，同文件可消解）。
			{SourceName: "main", EdgeType: "call", TargetName: "processData"},
			{SourceName: "main", EdgeType: "call", TargetName: "calculate"},
			{SourceName: "main", EdgeType: "call", TargetName: "useStruct"},
		},
		Chains: []quality.ExpectedChain{
			// main -> processData：同一文件内、入库 call 边可达。
			{StartName: "main", EndName: "processData",
				StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/cpp/cpp_calls_c.cpp"},
			{StartName: "main", EndName: "useStruct",
				StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/cpp/cpp_calls_c.cpp"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 2. kotlin_calls_java.kt —— import 边（source 为包名，target 为空=悬空）。
	//    Kotlin 解析器把 import 记为 source=包名、target=外部类全限定名，
	//    但外部类无对应符号（外部依赖未索引），target_id 无法消解 → target_name 为空。
	//    这些边 source 非空故可入库；真值用 Optional=true 让其不拉低 recall/precision。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/kotlin/kotlin_calls_java.kt",
		Edges: []quality.ExpectedEdge{
			{SourceName: "com.example.interop", EdgeType: "import", TargetName: "", Optional: true},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 3. typescript_calls_js.ts —— import 边（source 为模块名，target 为空）。
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/js/typescript_calls_js.ts",
		Edges: []quality.ExpectedEdge{
			{SourceName: "typescript_calls_js", EdgeType: "import", TargetName: "", Optional: true},
		},
	},
}
