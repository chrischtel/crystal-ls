<div align="center">

# 💎 Crystal Language Server (crystal-ls)

*A _**highly experimental**_ implementation of the Language Server Protocol (LSP) for the Crystal programming language, written in Go.*

[![Latest Release](https://img.shields.io/github/v/release/chrischtel/gocry?style=flat-square&logo=github&label=Stable&v=1)](https://github.com/chrischtel/gocry/releases/latest)
[![Latest Dev](https://img.shields.io/badge/dev-latest--dev-orange?style=flat-square&logo=github&v=1)](https://github.com/chrischtel/gocry/releases/tag/latest-dev)
[![Go Version](https://img.shields.io/github/go-mod/go-version/chrischtel/gocry?style=flat-square&logo=go&v=1)](https://golang.org/)
[![License](https://img.shields.io/github/license/chrischtel/gocry?style=flat-square&v=1)](LICENSE)

</div>

---

## ✨ Features



## 🗺️ Roadmap
The project is in its early stages. The immediate focus is on establishing a solid foundation by implementing core LSP functionalities.

---

## 📦 Installation

<div align="center">

### Quick Download

| Platform | Stable | Development |
|----------|--------|-------------|
| **Linux (x64)** | [📥 Download](https://github.com/chrischtel/crystal-ls/releases/latest/download/crystal-ls-linux-amd64.tar.gz) | [🔧 Download](https://github.com/chrischtel/crystal-ls/releases/download/latest-dev/crystal-ls-linux-amd64.tar.gz) |
| **Linux (ARM64)** | [📥 Download](https://github.com/chrischtel/crystal-ls/releases/latest/download/crystal-ls-linux-arm64.tar.gz) | [🔧 Download](https://github.com/chrischtel/crystal-ls/releases/download/latest-dev/crystal-ls-linux-arm64.tar.gz) |
| **macOS (Intel)** | [📥 Download](https://github.com/chrischtel/crystal-ls/releases/latest/download/crystal-ls-darwin-amd64.tar.gz) | [🔧 Download](https://github.com/chrischtel/crystal-ls/releases/download/latest-dev/crystal-ls-darwin-amd64.tar.gz) |
| **macOS (Apple Silicon)** | [📥 Download](https://github.com/chrischtel/crystal-ls/releases/latest/download/crystal-ls-darwin-arm64.tar.gz) | [🔧 Download](https://github.com/chrischtel/crystal-ls/releases/download/latest-dev/crystal-ls-darwin-arm64.tar.gz) |
| **Windows (x64)** | [📥 Download](https://github.com/chrischtel/crystal-ls/releases/latest/download/crystal-ls-windows-amd64.exe.zip) | [🔧 Download](https://github.com/chrischtel/crystal-ls/releases/download/latest-dev/crystal-ls-windows-amd64.exe.zip) |

</div>

### 📋 Installation Steps

1. **Download** the appropriate archive for your platform
2. **Extract** the archive:
   ```bash
   # Linux/macOS
   tar -xzf crystal-ls-*.tar.gz
   
   # Windows
   unzip crystal-ls-*.zip
   ```
3. **Make executable** (Linux/macOS only):
   ```bash
   chmod +x crystal-ls-*
   ```
4. **Add to PATH** (recommended):
   ```bash
   # Linux/macOS
   sudo mv crystal-ls-* /usr/local/bin/crystal-ls
   
   # Windows - move to a directory in your PATH
   ```

### 🔧 VS Code Setup

1. Install the [Crystal Language extension](https://marketplace.visualstudio.com/items?itemName=crystal-lang-tools.crystal-lang)
2. Configure the language server path in VS Code settings:
   ```json
   {
     "crystal-lang.server": "/path/to/crystal-ls"
   }
   ```

---

## 🛠️ Development

### Building from Source

---

## 🛠️ Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/chrischtel/crystal-ls.git
cd crystal-ls

# Build the language server
go build -o crystal-ls main.go

# Run tests
go test ./...

# Check version
./crystal-ls --version
```

### Project Structure

```
crystal-ls/
├── main.go                 # Entry point
├── internal/
│   └── lsp/
│       ├── server.go       # LSP server implementation
│       ├── analyzer.go     # Code analysis engine
│       ├── lexer.go        # Crystal lexer
│       ├── types.go        # LSP protocol types
│       └── crystal_tool.go # Crystal tool integration
├── examples/               # Example Crystal files
└── .github/workflows/      # CI/CD pipelines
```

### Release Types

| Type | Trigger | Format | Purpose |
|------|---------|--------|---------|
| **Stable** | Manual tag | `v0.1.0` | Production releases |
| **Development** | Push to develop | `latest-dev` (updated in-place) | Latest features from develop branch |

---

## 🤝 Contributing

We welcome contributions! Here's how you can help:

- 🐛 **Report bugs** - Open an issue with reproduction steps
- 💡 **Suggest features** - Share your ideas for improvements
- 🔧 **Submit PRs** - Fix bugs or implement new features
- 📚 **Improve docs** - Help make the documentation better
- 🧪 **Test prereleases** - Try nightly builds and report issues

### Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run tests: `go test ./...`
5. Commit: `git commit -m "Add amazing feature"`
6. Push: `git push origin feature/amazing-feature`
7. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Crystal Language](https://crystal-lang.org/) - The amazing language this server supports
- [LSP Specification](https://microsoft.github.io/language-server-protocol/) - The protocol standard
- [VS Code Crystal Extension](https://marketplace.visualstudio.com/items?itemName=crystal-lang-tools.crystal-lang) - Official Crystal support for VS Code

---

<div align="center">

**Built with 💎 for the Crystal community**

[Website](https://crystal-lang.org/) • [Documentation](https://crystal-lang.org/docs/) • [Community](https://crystal-lang.org/community/)

</div>
