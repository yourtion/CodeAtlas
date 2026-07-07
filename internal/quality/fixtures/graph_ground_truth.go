// Package fixtures 存放评估真值（ground truth）。
//
// 真值来源：从 tests/integration/call_analysis_fixtures_test.go 里散落的
// expectedXxxCalls/expectedXxxImports/expectedXxxFrameworks 列表系统化迁移而来。
//
// 每条 Edge 是 (SourceName, EdgeType, TargetName) 三元组：
//   - SourceName 取自 fixture 文件里包含该调用的函数/方法名（读 fixture 确认）。
//   - 标准库/外部 API（如 strlen、malloc、printf、console、Promise）标 Optional=true。
//
// Chains 是端到端调用链，用于 call_chain_connectivity 指标。
package fixtures

import "github.com/yourtionguo/CodeAtlas/internal/quality"

// CallAnalysisGroundTruth 是 call_analysis fixture 集的依赖图真值。
var CallAnalysisGroundTruth = []quality.GraphGroundTruth{
	// ──────────────────────────────────────────────────────────────
	// 1. cpp_calls_c.cpp —— C++ 调用 C 函数
	// 来源：call_analysis_fixtures_test.go:39 的 expectedCCalls
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/cpp/cpp_calls_c.cpp",
		Edges: []quality.ExpectedEdge{
			// 构造函数 CWrapper::CWrapper
			{SourceName: "CWrapper::CWrapper", EdgeType: "call", TargetName: "c_init"},
			// 析构函数 CWrapper::~CWrapper
			{SourceName: "CWrapper::~CWrapper", EdgeType: "call", TargetName: "c_free"},
			{SourceName: "CWrapper::~CWrapper", EdgeType: "call", TargetName: "c_cleanup"},
			// CWrapper::processData
			{SourceName: "CWrapper::processData", EdgeType: "call", TargetName: "strlen", Optional: true},
			{SourceName: "CWrapper::processData", EdgeType: "call", TargetName: "malloc", Optional: true},
			{SourceName: "CWrapper::processData", EdgeType: "call", TargetName: "strcpy", Optional: true},
			{SourceName: "CWrapper::processData", EdgeType: "call", TargetName: "c_process_string"},
			// CWrapper::calculate
			{SourceName: "CWrapper::calculate", EdgeType: "call", TargetName: "c_add"},
			{SourceName: "CWrapper::calculate", EdgeType: "call", TargetName: "c_multiply"},
			// CWrapper::useStruct
			{SourceName: "CWrapper::useStruct", EdgeType: "call", TargetName: "c_init_struct"},
			{SourceName: "CWrapper::useStruct", EdgeType: "call", TargetName: "c_process_struct"},
			{SourceName: "CWrapper::useStruct", EdgeType: "call", TargetName: "c_free_struct"},
			// 自由函数 processCData
			{SourceName: "processCData", EdgeType: "call", TargetName: "printf", Optional: true},
			{SourceName: "processCData", EdgeType: "call", TargetName: "c_log_message"},
			{SourceName: "processCData", EdgeType: "call", TargetName: "c_validate_input"},
			// main
			{SourceName: "main", EdgeType: "call", TargetName: "printf", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			// main -> CWrapper::processData -> c_process_string
			{StartName: "main", EndName: "c_process_string", StartFile: "tests/fixtures/cpp/cpp_calls_c.cpp", EndFile: "tests/fixtures/c_library.h"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 2. simple_c_calls.m —— ObjC 调用 C 函数
	// 来源：call_analysis_fixtures_test.go:106 的 expectedCCalls
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/objc/simple_c_calls.m",
		Edges: []quality.ExpectedEdge{
			// SimpleWrapper::addNumbers:and:
			{SourceName: "SimpleWrapper::addNumbers:and:", EdgeType: "call", TargetName: "c_add"},
			// SimpleWrapper::logMessage:
			{SourceName: "SimpleWrapper::logMessage:", EdgeType: "call", TargetName: "c_log"},
			{SourceName: "SimpleWrapper::logMessage:", EdgeType: "call", TargetName: "printf", Optional: true},
			{SourceName: "SimpleWrapper::logMessage:", EdgeType: "call", TargetName: "strlen", Optional: true},
			// 自由函数 processWithC
			{SourceName: "processWithC", EdgeType: "call", TargetName: "c_log"},
			{SourceName: "processWithC", EdgeType: "call", TargetName: "printf", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			{StartName: "processWithC", EndName: "c_log", StartFile: "tests/fixtures/objc/simple_c_calls.m", EndFile: "tests/fixtures/objc/simple_c_calls.m"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 3. simple_cpp_calls.mm —— ObjC++ 调用 C++ 函数
	// 来源：call_analysis_fixtures_test.go:169 的 expectedCppCalls
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/objc/simple_cpp_calls.mm",
		Edges: []quality.ExpectedEdge{
			// CppBridge::addNumbers:and: -> CppHelper::add
			{SourceName: "CppBridge::addNumbers:and:", EdgeType: "call", TargetName: "add"},
			// CppBridge::getCppMessage -> CppHelper::getMessage, c_str
			{SourceName: "CppBridge::getCppMessage", EdgeType: "call", TargetName: "getMessage"},
			{SourceName: "CppBridge::getCppMessage", EdgeType: "call", TargetName: "c_str", Optional: true},
			// CppBridge::multiplyNumbers:and: -> cpp_multiply
			{SourceName: "CppBridge::multiplyNumbers:and:", EdgeType: "call", TargetName: "cpp_multiply"},
			// processWithCpp -> std::vector::push_back
			{SourceName: "processWithCpp", EdgeType: "call", TargetName: "push_back", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			{StartName: "CppBridge::addNumbers:and:", EndName: "add", StartFile: "tests/fixtures/objc/simple_cpp_calls.mm", EndFile: "tests/fixtures/objc/simple_cpp_calls.mm"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 4. kotlin_calls_java.kt —— Kotlin 调用 Java
	// 来源：call_analysis_fixtures_test.go:230 expectedJavaImports + :241 expectedJavaCalls
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/kotlin/kotlin_calls_java.kt",
		Edges: []quality.ExpectedEdge{
			// —— imports ——
			{SourceName: "KotlinJavaInterop", EdgeType: "import", TargetName: "java.util.ArrayList", Optional: true},
			{SourceName: "KotlinJavaInterop", EdgeType: "import", TargetName: "java.util.HashMap", Optional: true},
			{SourceName: "KotlinJavaInterop", EdgeType: "import", TargetName: "java.util.Date", Optional: true},
			{SourceName: "KotlinJavaInterop", EdgeType: "import", TargetName: "java.text.SimpleDateFormat", Optional: true},
			{SourceName: "KotlinJavaInterop", EdgeType: "import", TargetName: "java.io.File", Optional: true},
			{SourceName: "KotlinJavaInterop", EdgeType: "import", TargetName: "java.io.FileReader", Optional: true},
			{SourceName: "KotlinJavaInterop", EdgeType: "import", TargetName: "java.io.BufferedReader", Optional: true},
			// —— calls ——
			// useJavaArrayList
			{SourceName: "useJavaArrayList", EdgeType: "call", TargetName: "ArrayList", Optional: true},
			{SourceName: "useJavaArrayList", EdgeType: "call", TargetName: "add", Optional: true},
			{SourceName: "useJavaArrayList", EdgeType: "call", TargetName: "get", Optional: true},
			// useJavaHashMap
			{SourceName: "useJavaHashMap", EdgeType: "call", TargetName: "HashMap", Optional: true},
			{SourceName: "useJavaHashMap", EdgeType: "call", TargetName: "put", Optional: true},
			// useJavaDate
			{SourceName: "useJavaDate", EdgeType: "call", TargetName: "Date", Optional: true},
			{SourceName: "useJavaDate", EdgeType: "call", TargetName: "SimpleDateFormat", Optional: true},
			{SourceName: "useJavaDate", EdgeType: "call", TargetName: "format", Optional: true},
			// readJavaFile
			{SourceName: "readJavaFile", EdgeType: "call", TargetName: "File", Optional: true},
			{SourceName: "readJavaFile", EdgeType: "call", TargetName: "exists", Optional: true},
			{SourceName: "readJavaFile", EdgeType: "call", TargetName: "FileReader", Optional: true},
			{SourceName: "readJavaFile", EdgeType: "call", TargetName: "BufferedReader", Optional: true},
			{SourceName: "readJavaFile", EdgeType: "call", TargetName: "readLine", Optional: true},
			{SourceName: "readJavaFile", EdgeType: "call", TargetName: "close", Optional: true},
			// useJavaString
			{SourceName: "useJavaString", EdgeType: "call", TargetName: "length", Optional: true},
			{SourceName: "useJavaString", EdgeType: "call", TargetName: "toUpperCase", Optional: true},
			{SourceName: "useJavaString", EdgeType: "call", TargetName: "substring", Optional: true},
			// useJavaSystem
			{SourceName: "useJavaSystem", EdgeType: "call", TargetName: "currentTimeMillis", Optional: true},
			{SourceName: "useJavaSystem", EdgeType: "call", TargetName: "getProperty", Optional: true},
			// JavaStaticCalls.callStaticMethods
			{SourceName: "callStaticMethods", EdgeType: "call", TargetName: "max", Optional: true},
			{SourceName: "callStaticMethods", EdgeType: "call", TargetName: "sqrt", Optional: true},
			{SourceName: "callStaticMethods", EdgeType: "call", TargetName: "parseInt", Optional: true},
			{SourceName: "callStaticMethods", EdgeType: "call", TargetName: "toHexString", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			// readJavaFile -> FileReader -> BufferedReader (file I/O 链)
			{StartName: "readJavaFile", EndName: "BufferedReader", StartFile: "tests/fixtures/kotlin/kotlin_calls_java.kt", EndFile: "tests/fixtures/kotlin/kotlin_calls_java.kt"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 5. swift_calls_objc.swift —— Swift 调用 ObjC 框架
	// 来源：call_analysis_fixtures_test.go:336 expectedFrameworks + :342 expectedObjCCalls
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/swift/swift_calls_objc.swift",
		Edges: []quality.ExpectedEdge{
			// —— framework imports ——
			{SourceName: "SwiftViewController", EdgeType: "import", TargetName: "Foundation", Optional: true},
			{SourceName: "SwiftViewController", EdgeType: "import", TargetName: "UIKit", Optional: true},
			// —— calls ——
			// processString
			{SourceName: "processString", EdgeType: "call", TargetName: "NSString", Optional: true},
			{SourceName: "processString", EdgeType: "call", TargetName: "length", Optional: true},
			{SourceName: "processString", EdgeType: "call", TargetName: "uppercased", Optional: true},
			// processArray
			{SourceName: "processArray", EdgeType: "call", TargetName: "NSArray", Optional: true},
			{SourceName: "processArray", EdgeType: "call", TargetName: "count", Optional: true},
			{SourceName: "processArray", EdgeType: "call", TargetName: "firstObject", Optional: true},
			// processDictionary
			{SourceName: "processDictionary", EdgeType: "call", TargetName: "NSDictionary", Optional: true},
			{SourceName: "processDictionary", EdgeType: "call", TargetName: "object", Optional: true},
			// setupNotifications
			{SourceName: "setupNotifications", EdgeType: "call", TargetName: "addObserver", Optional: true},
			// savePreferences
			{SourceName: "savePreferences", EdgeType: "call", TargetName: "set", Optional: true},
			{SourceName: "savePreferences", EdgeType: "call", TargetName: "synchronize", Optional: true},
			// checkFile
			{SourceName: "checkFile", EdgeType: "call", TargetName: "fileExists", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			{StartName: "processArray", EndName: "NSArray", StartFile: "tests/fixtures/swift/swift_calls_objc.swift", EndFile: "tests/fixtures/swift/swift_calls_objc.swift"},
		},
	},

	// ──────────────────────────────────────────────────────────────
	// 6. typescript_calls_js.ts —— TypeScript 调用 JavaScript
	// 来源：call_analysis_fixtures_test.go:452 expectedJSImports + :459 expectedJSCalls
	// ──────────────────────────────────────────────────────────────
	{
		FixtureFile: "tests/fixtures/js/typescript_calls_js.ts",
		Edges: []quality.ExpectedEdge{
			// —— imports ——
			{SourceName: "TypeScriptComponent", EdgeType: "import", TargetName: "./legacy-module.js", Optional: true},
			{SourceName: "TypeScriptComponent", EdgeType: "import", TargetName: "./utils.js", Optional: true},
			{SourceName: "TypeScriptComponent", EdgeType: "import", TargetName: "./default-export.js", Optional: true},
			// —— calls ——
			// useJavaScriptGlobals -> console / setTimeout / setInterval / clearInterval / Promise
			{SourceName: "useJavaScriptGlobals", EdgeType: "call", TargetName: "console", Optional: true},
			{SourceName: "useJavaScriptGlobals", EdgeType: "call", TargetName: "setTimeout", Optional: true},
			{SourceName: "useJavaScriptGlobals", EdgeType: "call", TargetName: "setInterval", Optional: true},
			{SourceName: "useJavaScriptGlobals", EdgeType: "call", TargetName: "clearInterval", Optional: true},
			{SourceName: "useJavaScriptGlobals", EdgeType: "call", TargetName: "Promise", Optional: true},
			// useJavaScriptFetch -> fetch
			{SourceName: "useJavaScriptFetch", EdgeType: "call", TargetName: "fetch", Optional: true},
			// useJavaScriptLocalStorage -> localStorage
			{SourceName: "useJavaScriptLocalStorage", EdgeType: "call", TargetName: "localStorage", Optional: true},
			// useJavaScriptRequire -> require
			{SourceName: "useJavaScriptRequire", EdgeType: "call", TargetName: "require", Optional: true},
		},
		Chains: []quality.ExpectedChain{
			// useJavaScriptGlobals -> setTimeout (定时器链)
			{StartName: "useJavaScriptGlobals", EndName: "setTimeout", StartFile: "tests/fixtures/js/typescript_calls_js.ts", EndFile: "tests/fixtures/js/typescript_calls_js.ts"},
		},
	},
}
