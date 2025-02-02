# jujuid (You-You-ID)

## 🎉 Some random project during fosdem 2025

jujuid is a playful UUID-to-Name translator that transforms boring UUIDs into
memorable, human-readable identifiers.

![Demo](./assets/demo.gif)

### Fosdem is a wrap 🏁

That marks the KFC release!! 🐔 v0.1.1-kfc 

#### What is jujuid?

jujuid is a lightweight command-line tool that replaces UUID strings with
generated human-readable names. Perfect for making log files, debug output, and
system traces more readable and fun!

### Features

- Automatically generates memorable names for UUIDs
- Persistent mapping of UUIDs to names
- Simple stdin/stdout interface
- Signal handling for graceful termination

### Quick Example

```bash
echo "Processing request from b56d2ce4-484d-49bb-89cb-da4517df6c66" | ./jujuid
# Output: Processing request from [UUID: Mr John Smith]
```

### Getting Started

#### Prerequisites

• Go 1.20+

#### Installation

go get github.com/jzandbergen/jujuid
go build

### Usage

cat logfile.log | ./jujuid
# or
./jujuid < input.txt

### Current Limitations

• In-memory UUID mapping (resets on exit)
• Basic UUID detection
• Minimal error handling

### Roadmap

- There is no roadmap.

### Developed at FOSDEM 2025 🍺🖥️

Crafted with ❤️ during hacking sessions!


