/// The tool's pure logic — no argument parsing, no I/O — so it is provable with
/// `swift test` alone. The executable target is a thin shell that parses the command
/// line and delegates here.
public enum Greeter {
    /// Build the greeting line for a name. Shouting upper-cases the whole line.
    public static func greeting(for name: String, shout: Bool = false) -> String {
        let line = "Hello, \(name)!"
        return shout ? line.uppercased() : line
    }
}
