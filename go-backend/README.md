# Humanity Galaxy

A decade-resilient, flat-file, infinite image archive built to outlive databases, frameworks, and hype cycles.

---

## Motivation

Most systems rot because they depend on things that age badly: databases, services, managed platforms, humans.  
This project is built with one rule that matters if you want it alive in 2045:

**Simplicity over Complexity.**

If every database dies, every cloud shuts down, and only a hard drive remains, the folder structure alone must be enough to reconstruct the entire Galaxy.

Humanity Galaxy is a permanent, zoomable archive of human presence, optimized for:
- Longevity over features
- Files over services
- Determinism over convenience

If it is not boring, it is not reliable.

---

## Quick Start

### Requirements
- Go 1.22+
- Any POSIX filesystem
- Zero external services required for local mode

### Clone & Run
```bash
git clone https://github.com/yourname/humanity-galaxy.git
cd humanity-galaxy/backend
go run main.go
```

### Minimal Test Setup
```bash
mkdir -p storage/20/0
cp test.webp storage/20/0/0.webp
```

Run four simulated uploads and verify that:
```
storage/19/0/0.webp
```
is generated automatically.

---

## Usage

### Storage Model (Flat-File, Quadtree)

Tiles are addressed by zoom level and coordinates:

```
/tiles/{z}/{x}/{y}.webp
```

Local filesystem layout:
```
/storage
 ├── /20
 │   ├── /1024
 │   │   ├── 2048.webp
 │   │   └── 2049.webp
 │   ├── /1025
 │   │   └── 2048.webp
 ├── /19
 │   └── /512
 │       └── 1024.webp
 └── ...
```

---

## The “Decades” Algorithm (Group of 4)

For any tile `(Z, X, Y)`:

- Check:
  - `(X+1, Y)`
  - `(X, Y+1)`
  - `(X+1, Y+1)`

If all exist:
1. Merge into 512×512 canvas
2. Downscale to 256×256
3. Save to `Z-1, X/2, Y/2`
4. Repeat upward until Z=0 or merge fails

---

## Contributing

This project values:
- Determinism
- Flat-file durability
- Decade-scale thinking

Rejected immediately:
- Mandatory databases
- Stateful services
- Trend-driven abstractions

If your change cannot survive without a database, it does not belong here.

---

## Technology Stack

- **Language:** Go  
- **Image Engine:** Native Go + imaging  
- **Storage:** Quadtree filesystem  
- **Permanent Archive:** Arweave  
- **Database (Optional):** SQLite  

---

## Reliability & Cost Model

| Component | Strategy | Longevity |
|---------|--------|----------|
| Server | Any VPS or PC | High |
| Storage | Flat files + Arweave | Extreme |
| Backend | Single Go binary | Extreme |
| Access | Static URLs | Extreme |
| Maintenance | Domain renewal | Human risk |

---

## Final Test Recommendation

1. Create 4 tiles at `Z=20`
2. Verify merge at `Z=19`
3. Serve files statically
4. Zoom out in Three.js

If this works, the system is stable for decades.
