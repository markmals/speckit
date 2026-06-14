import Foundation
import SwiftData

/// The on-disk shape of a to-do, owned by the persistence layer (not the domain)
/// and keyed by the domain's stable id. Keeping the stored shape separate keeps
/// the domain pure (no SwiftData import in the Core) and lets the cache evolve its
/// own schema.
@Model
final class TodoRecord {
    @Attribute(.unique) var id: UUID
    var label: String
    var isDone: Bool

    init(id: UUID, label: String, isDone: Bool) {
        self.id = id
        self.label = label
        self.isDone = isDone
    }
}
