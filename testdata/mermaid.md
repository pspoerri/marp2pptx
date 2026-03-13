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

## Class Diagram

```mermaid
classDiagram
    class Animal {
        +String name
        +int age
        +makeSound()
    }
    class Dog {
        +String breed
        +bark()
    }
    class Cat {
        +bool indoor
        +meow()
    }
    Animal <|-- Dog
    Animal <|-- Cat
```

---

## Class Diagram with Relationships

```mermaid
classDiagram
    class Vehicle {
        +String make
        +String model
        +start()
    }
    class Engine {
        +int horsepower
        +run()
    }
    class Wheel {
        +int size
    }
    class Driver {
        +String name
        +drive()
    }
    Vehicle *-- Engine : has
    Vehicle o-- Wheel : has 4
    Driver --> Vehicle : drives
    Vehicle ..|> Drivable
```

---

## State Diagram

```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Processing : start
    Processing --> Done : finish
    Processing --> Error : fail
    Error --> Idle : retry
    Done --> [*]
```

---

## State Diagram (Traffic Light)

```mermaid
stateDiagram-v2
    [*] --> Red
    Red --> Green : timer
    Green --> Yellow : timer
    Yellow --> Red : timer
```

---

## State Diagram (Movement)

```mermaid
stateDiagram-v2
    [*] --> Still
    Still --> [*]
    Still --> Moving
    Moving --> Still
    Moving --> Crash
    Crash --> [*]
```

---

## State Diagram (Aliases)

```mermaid
stateDiagram-v2
    state "First State" as First
    state "Named Composite" as Named
    state "Simple State" as Simple
    [*] --> First
    First --> Named
    Named --> Simple
    Simple --> [*]
```

---

## User Journey

```mermaid
journey
    title My Working Day
    section Morning
        Wake up: 3: Me
        Make coffee: 5: Me
        Commute: 2: Me, Bus
    section Work
        Code review: 4: Me
        Write code: 5: Me
        Lunch: 4: Me, Team
    section Evening
        Commute home: 2: Me, Bus
        Dinner: 5: Me, Family
```

---

## User Journey (Online Shopping)

```mermaid
journey
    title Online Shopping Experience
    section Browse
        Search product: 4: Customer
        View details: 3: Customer
        Read reviews: 4: Customer
    section Purchase
        Add to cart: 5: Customer
        Checkout: 3: Customer
        Enter payment: 2: Customer
    section Delivery
        Track package: 4: Customer
        Receive delivery: 5: Customer, Courier
```

---

## Entity Relationship Diagram

```mermaid
erDiagram
    CUSTOMER ||--o{ ORDER : places
    ORDER ||--|{ LINE-ITEM : contains
    CUSTOMER }|..|{ DELIVERY-ADDRESS : uses
    ORDER }o--|| PAYMENT : requires
```

---

## ER Diagram with Attributes

```mermaid
erDiagram
    STUDENT {
        int id PK
        string name
        string email
    }
    COURSE {
        int id PK
        string title
        int credits
    }
    ENROLLMENT {
        int id PK
        date enrolled_at
        string grade
    }
    STUDENT ||--o{ ENROLLMENT : has
    COURSE ||--o{ ENROLLMENT : has
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
