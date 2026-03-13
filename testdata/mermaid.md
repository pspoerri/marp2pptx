---
marp: true
---

# Mermaid Diagrams

---

## Flowchart (LR)

```mermaid
graph LR
    A[Start] --> B{Decision}
    B -->|Yes| C[OK]
    B -->|No| D[Cancel]
```

---

## Flowchart (TD) with Shapes

```mermaid
graph TD
    A([Start]) --> B[Process]
    B --> C{Decision}
    C -->|Yes| D[(Database)]
    C -->|No| E((End))
```

---

## Sequence Diagram

```mermaid
sequenceDiagram
    Alice->>Bob: Hello Bob
    Bob-->>Alice: Hi Alice
    Alice->>Bob: How are you?
    Bob-->>Alice: Great!
```

---

## Sequence with Participants

```mermaid
sequenceDiagram
    participant Client
    participant Server
    participant DB as Database
    Client->>Server: HTTP Request
    Server->>DB: Query
    DB-->>Server: Results
    Server-->>Client: HTTP Response
```

---

## Dotted and Thick Edges

```mermaid
graph LR
    A -.-> B
    B ==> C
    C --- D
    D --> A
```

---

## Mixed Content

Some text before the diagram.

```mermaid
graph TD
    A --> B
```

And some text after.
