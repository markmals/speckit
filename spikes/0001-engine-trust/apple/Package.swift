// swift-tools-version: 6.0
import PackageDescription

let package = Package(
	name: "Todo",
	targets: [
		.target(name: "Todo"),
		.testTarget(name: "TodoTests", dependencies: ["Todo"]),
	]
)
