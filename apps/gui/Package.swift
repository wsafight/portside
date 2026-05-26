// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "Portside",
    platforms: [
        .macOS(.v15)
    ],
    products: [
        .executable(name: "Portside", targets: ["Portside"])
    ],
    targets: [
        .executableTarget(name: "Portside")
    ]
)
