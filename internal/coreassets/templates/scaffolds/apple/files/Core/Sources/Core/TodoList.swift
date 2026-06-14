import Observation

/// The list's observable state. State management is Swift Observation: AppKit /
/// UIKit views observe this model, while the model itself stays UI-free so it can
/// be proven headlessly in the Core package.
@Observable
public final class TodoList {
    public private(set) var items: [Todo]

    public init(items: [Todo] = []) { self.items = items }

    /// Append a new item by label.
    public func add(_ label: String) { items.append(Todo(label: label)) }

    /// Toggle the item with the given id, if present.
    public func toggle(_ id: Todo.ID) {
        guard let i = items.firstIndex(where: { $0.id == id }) else { return }
        items[i] = items[i].toggled()
    }
}
