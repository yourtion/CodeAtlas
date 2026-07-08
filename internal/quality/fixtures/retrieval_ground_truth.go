// 检索评估真值集。
//
// 覆盖：跨语言（3）+ 单语言（3）+ 多 repo（2）共 8 个 query。
// 每个 query 标注"真值相关符号"——即答案应包含的符号名。
//
// 符号名提取自 graph_ground_truth.go（Task 7）里各 fixture 的
// ExpectedEdge.SourceName / TargetName。仅取用户代码符号，
// 跳过 Optional=true 的标准库/外部 API（如 strlen、printf、console）。
//
// Repos 字段用 fixture 标识（文件名/语言库名）。最终 repoID 在 Task 10
// 集成索引时确定，本文件先以 fixture 标识占位。
package fixtures

import "github.com/yourtionguo/CodeAtlas/internal/quality"

// RetrievalGroundTruths 是检索评估真值集。
var RetrievalGroundTruths = []quality.RetrievalGroundTruth{
	// ──────────────────────────────────────────────────────────────
	// 跨语言 1：C++ 调用 C 函数（cpp_calls_c.cpp + C 库）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "C++ 如何调用 C 函数",
		RelevantSymbols: []string{
			"CWrapper::CWrapper",
			"CWrapper::~CWrapper",
			"CWrapper::processData",
			"CWrapper::calculate",
			"CWrapper::useStruct",
			"c_init",
			"c_free",
			"c_cleanup",
			"c_process_string",
			"c_add",
			"c_multiply",
			"c_init_struct",
			"c_process_struct",
			"c_free_struct",
			"c_log_message",
			"c_validate_input",
		},
		RelevantFiles: []string{
			"tests/fixtures/cpp/cpp_calls_c.cpp",
			"tests/fixtures/c_library.h",
		},
		Repos: []string{"cpp_calls_c.cpp", "c_library.h"},
	},

	// ──────────────────────────────────────────────────────────────
	// 跨语言 2：Kotlin 调用 Java（kotlin_calls_java.kt + Java stdlib）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "Kotlin 调用 Java 的哪些方法",
		RelevantSymbols: []string{
			"KotlinJavaInterop",
			"useJavaArrayList",
			"useJavaHashMap",
			"useJavaDate",
			"readJavaFile",
			"useJavaString",
			"useJavaSystem",
			"callStaticMethods",
		},
		RelevantFiles: []string{
			"tests/fixtures/kotlin/kotlin_calls_java.kt",
		},
		Repos: []string{"kotlin_calls_java.kt", "java-stdlib"},
	},

	// ──────────────────────────────────────────────────────────────
	// 跨语言 3：Swift 互操作 Objective-C（swift_calls_objc.swift）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "Swift 如何互操作 Objective-C",
		RelevantSymbols: []string{
			"SwiftViewController",
			"processString",
			"processArray",
			"processDictionary",
			"setupNotifications",
			"savePreferences",
			"checkFile",
		},
		RelevantFiles: []string{
			"tests/fixtures/swift/swift_calls_objc.swift",
		},
		Repos: []string{"swift_calls_objc.swift", "uikit"},
	},

	// ──────────────────────────────────────────────────────────────
	// 单语言 1：TypeScript 调用 JavaScript（typescript_calls_js.ts）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "JavaScript 模块导入了什么",
		RelevantSymbols: []string{
			"TypeScriptComponent",
			"useJavaScriptGlobals",
			"useJavaScriptFetch",
			"useJavaScriptLocalStorage",
			"useJavaScriptRequire",
		},
		RelevantFiles: []string{
			"tests/fixtures/js/typescript_calls_js.ts",
		},
		Repos: []string{"typescript_calls_js.ts"},
	},

	// ──────────────────────────────────────────────────────────────
	// 单语言 2：Objective-C 调用 C 函数（simple_c_calls.m）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "Objective-C 调用了哪些 C 函数",
		RelevantSymbols: []string{
			"SimpleWrapper::addNumbers:and:",
			"SimpleWrapper::logMessage:",
			"processWithC",
			"c_add",
			"c_log",
		},
		RelevantFiles: []string{
			"tests/fixtures/objc/simple_c_calls.m",
		},
		Repos: []string{"simple_c_calls.m"},
	},

	// ──────────────────────────────────────────────────────────────
	// 单语言 3：ObjC++ 桥接 C++（simple_cpp_calls.mm）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "ObjC++ 如何桥接 C++",
		RelevantSymbols: []string{
			"CppBridge::addNumbers:and:",
			"CppBridge::getCppMessage",
			"CppBridge::multiplyNumbers:and:",
			"processWithCpp",
			"add",
			"getMessage",
			"cpp_multiply",
		},
		RelevantFiles: []string{
			"tests/fixtures/objc/simple_cpp_calls.mm",
		},
		Repos: []string{"simple_cpp_calls.mm"},
	},

	// ──────────────────────────────────────────────────────────────
	// 多 repo 1：跨仓库符号检索（cpp + kotlin + swift）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "多仓库符号检索：各语言的互操作入口",
		RelevantSymbols: []string{
			"CWrapper::processData",
			"c_process_string",
			"useJavaArrayList",
			"readJavaFile",
			"processString",
			"processArray",
		},
		RelevantFiles: []string{
			"tests/fixtures/cpp/cpp_calls_c.cpp",
			"tests/fixtures/kotlin/kotlin_calls_java.kt",
			"tests/fixtures/swift/swift_calls_objc.swift",
		},
		Repos: []string{"cpp_calls_c.cpp", "kotlin_calls_java.kt", "swift_calls_objc.swift"},
	},

	// ──────────────────────────────────────────────────────────────
	// 多 repo 2：跨语言字符串处理（swift + kotlin）
	// ──────────────────────────────────────────────────────────────
	{
		Query: "跨语言字符串处理方法",
		RelevantSymbols: []string{
			"processString",
			"useJavaString",
		},
		RelevantFiles: []string{
			"tests/fixtures/swift/swift_calls_objc.swift",
			"tests/fixtures/kotlin/kotlin_calls_java.kt",
		},
		Repos: []string{"swift_calls_objc.swift", "kotlin_calls_java.kt"},
	},

	// ──────────────────────────────────────────────────────────────
	// 单语言 3（Java）：UserRepository 方法级检索
	// ──────────────────────────────────────────────────────────────
	{
		Query: "如何根据 ID 查找用户",
		RelevantSymbols: []string{
			"findById",
			"UserRepository",
		},
		RelevantFiles: []string{
			"tests/fixtures/java/src/main/java/com/example/myapp/repository/UserRepository.java",
		},
		Repos: []string{"UserRepository.java"},
	},
	// ──────────────────────────────────────────────────────────────
	// 单语言 4（Kotlin）：UserService 方法级检索
	// ──────────────────────────────────────────────────────────────
	{
		Query: "如何创建和删除用户",
		RelevantSymbols: []string{
			"createUser",
			"deleteUser",
			"UserService",
		},
		RelevantFiles: []string{
			"tests/fixtures/kotlin/src/main/kotlin/com/example/myapp/service/UserService.kt",
		},
		Repos: []string{"UserService.kt"},
	},
}
