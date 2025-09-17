# Function Mind Map

A React-based collapsible mind map visualization for function call hierarchies, inspired by NotebookLM's interface.

## Features

- 🌳 **Hierarchical Visualization**: Display function call relationships in a tree structure
- 🔄 **Collapsible Nodes**: Expand and collapse function call branches
- 🎨 **Dark Theme**: Clean, modern dark interface without gradients
- 🔍 **Zoom & Pan**: Navigate large function hierarchies with mouse controls
- 📱 **Interactive**: Click nodes to view detailed information
- 🎯 **Function Types**: Visual indicators for different types of functions (main, handlers, middleware, etc.)

## Getting Started

### Prerequisites

- Node.js (version 14 or higher)
- npm or yarn

### Installation

1. Navigate to the project directory:
   ```bash
   cd mind-map-react
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Start the development server:
   ```bash
   npm start
   ```

4. Open [http://localhost:3000](http://localhost:3000) to view it in the browser.

## Usage

### Controls

- **Click and Drag**: Pan around the mind map
- **Mouse Wheel**: Zoom in and out
- **Click Node**: Select and view function details
- **Click +/- Button**: Expand or collapse function branches
- **Reset Button**: Return to original zoom and position

### Data Format

The mind map expects data in the following format:

```json
[
  {
    "name": "main.main",
    "line": 9,
    "filePath": "main.go",
    "called": [
      {
        "name": "config.Load",
        "line": 9,
        "filePath": "internal\\config\\config.go"
      }
    ]
  }
]
```

### Customization

To use your own function map data:

1. Replace the `sampleData` in `src/App.js` with your function map JSON
2. Or modify the `useEffect` to fetch from your `functionmap.json` file

## Project Structure

```
src/
├── components/
│   ├── MindMap.js          # Main mind map component
│   ├── MindMap.css         # Mind map styling
│   └── Node.js             # Individual node component
├── App.js                  # Main application component
├── App.css                 # Application styling
├── index.js                # React entry point
└── index.css               # Global styles
```

## Function Types

The mind map automatically categorizes functions based on their names:

- **Main**: Entry point functions (purple)
- **Handler**: HTTP handlers and controllers (red)
- **Middleware**: Middleware functions (orange)
- **Config**: Configuration functions (green)
- **Router**: Routing functions (violet)
- **Function**: General functions (gray)

## Built With

- React 18
- CSS3
- SVG for graphics rendering
- No external dependencies for the mind map functionality

## License

This project is open source and available under the [MIT License](LICENSE).
