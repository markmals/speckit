/// A semantic version (major.minor.patch) — a small, pure value type, the kind of
/// thing a reusable package exposes. No framework dependency, so it's provable with
/// `swift test` alone.
public struct SemanticVersion: Sendable, Equatable, Comparable, CustomStringConvertible {
    public let major: Int
    public let minor: Int
    public let patch: Int

    public init(major: Int, minor: Int, patch: Int) {
        self.major = major
        self.minor = minor
        self.patch = patch
    }

    /// Parse `"major.minor.patch"` (e.g. `"1.2.3"`). Returns nil if the shape is wrong.
    public init?(parsing string: String) {
        let parts = string.split(separator: ".", omittingEmptySubsequences: false)
        guard parts.count == 3,
            let major = Int(parts[0]),
            let minor = Int(parts[1]),
            let patch = Int(parts[2])
        else { return nil }
        self.init(major: major, minor: minor, patch: patch)
    }

    public var description: String { "\(major).\(minor).\(patch)" }

    /// Ordering is numeric per component — so 1.10.0 is newer than 1.2.0, the trap a
    /// naive lexical compare falls into.
    public static func < (lhs: SemanticVersion, rhs: SemanticVersion) -> Bool {
        (lhs.major, lhs.minor, lhs.patch) < (rhs.major, rhs.minor, rhs.patch)
    }
}
