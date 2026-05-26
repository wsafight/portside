import SwiftUI

@main
struct PortsideApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}

struct ContentView: View {
    var body: some View {
        Text("Portside")
            .frame(minWidth: 640, minHeight: 420)
    }
}
