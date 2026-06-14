import Foundation

/// A to-do item — the pure, headless domain type the example scenario proves.
/// No AppKit, no UIKit, no SwiftData: the Core package builds and tests with
/// `swift test` alone, which is exactly what `specify verify` runs.
public struct Todo: Sendable, Equatable, Identifiable {
    public let id: UUID
    public var label: String
    public var isDone: Bool

    public init(id: UUID = UUID(), label: String, isDone: Bool = false) {
        self.id = id
        self.label = label
        self.isDone = isDone
    }

    /// Toggling flips completion — the invariant the example scenario pins.
    public func toggled() -> Todo {
        Todo(id: id, label: label, isDone: !isDone)
    }
}
