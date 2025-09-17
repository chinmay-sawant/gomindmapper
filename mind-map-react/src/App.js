import React, { useState, useEffect } from 'react';
import MindMap from './components/MindMap';
import './App.css';

// Sample function map data (in a real app, this would be loaded from your functionmap.json)
const sampleData = [
  {
    "name": "handlers.handleFillPDF",
    "line": 168,
    "filePath": "gopdfsuit\\internal\\handlers\\handlers.go",
    "called": [
      {
        "name": "pdf.FillPDFWithXFDF",
        "line": 718,
        "filePath": "gopdfsuit\\internal\\pdf\\xfdf.go"
      }
    ]
  },
  {
    "name": "handlers.handleGenerateTemplatePDF",
    "line": 157,
    "filePath": "gopdfsuit\\internal\\handlers\\handlers.go",
    "called": [
      {
        "name": "pdf.GenerateTemplatePDF",
        "line": 15,
        "filePath": "gopdfsuit\\internal\\pdf\\generator.go"
      }
    ]
  },
  {
    "name": "handlers.handleMergePDFs",
    "line": 222,
    "filePath": "gopdfsuit\\internal\\handlers\\handlers.go",
    "called": [
      {
        "name": "pdf.MergePDFs",
        "line": 13,
        "filePath": "gopdfsuit\\internal\\pdf\\merge.go"
      }
    ]
  },
  {
    "name": "handlers.handlehtmlToImage",
    "line": 308,
    "filePath": "gopdfsuit\\internal\\handlers\\handlers.go",
    "called": [
      {
        "name": "pdf.ConvertHTMLToImage",
        "line": 59,
        "filePath": "gopdfsuit\\internal\\pdf\\pdf.go"
      }
    ]
  },
  {
    "name": "handlers.handlehtmlToPDF",
    "line": 266,
    "filePath": "gopdfsuit\\internal\\handlers\\handlers.go",
    "called": [
      {
        "name": "pdf.ConvertHTMLToPDF",
        "line": 21,
        "filePath": "gopdfsuit\\internal\\pdf\\pdf.go"
      }
    ]
  },
  {
    "name": "main.convertToImage",
    "line": 140,
    "filePath": "gochromedp\\cmd\\gochromedp\\main.go",
    "called": [
      {
        "name": "gochromedp.ConvertURLToImage",
        "line": 268,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      },
      {
        "name": "gochromedp.ConvertHTMLToImage",
        "line": 201,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      }
    ]
  },
  {
    "name": "main.convertToPDF",
    "line": 108,
    "filePath": "gochromedp\\cmd\\gochromedp\\main.go",
    "called": [
      {
        "name": "gochromedp.ConvertURLToPDF",
        "line": 128,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      },
      {
        "name": "gochromedp.ConvertHTMLToPDF",
        "line": 52,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      }
    ]
  },
  {
    "name": "main.findFunctions",
    "line": 62,
    "filePath": "cmd\\main.go",
    "called": [
      {
        "name": "analyzer.FindFunctionBody",
        "line": 26,
        "filePath": "cmd\\analyzer\\utils.go"
      },
      {
        "name": "analyzer.FindCalls",
        "line": 48,
        "filePath": "cmd\\analyzer\\utils.go"
      }
    ]
  },
  {
    "name": "main.main",
    "line": 8,
    "filePath": "gopdfsuit\\cmd\\gopdfsuit\\main.go",
    "called": [
      {
        "name": "handlers.RegisterRoutes",
        "line": 77,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      }
    ]
  },
  {
    "name": "main.main",
    "line": 18,
    "filePath": "cmd\\main.go",
    "called": [
      {
        "name": "analyzer.GetModule",
        "line": 11,
        "filePath": "cmd\\analyzer\\utils.go"
      },
      {
        "name": "analyzer.CreateJsonFile",
        "line": 10,
        "filePath": "cmd\\analyzer\\fileops.go"
      }
    ]
  },
  {
    "name": "main.main",
    "line": 9,
    "filePath": "EmployeeApp\\main.go",
    "called": [
      {
        "name": "config.Load",
        "line": 9,
        "filePath": "EmployeeApp\\internal\\config\\config.go"
      },
      {
        "name": "routes.SetupRouter",
        "line": 10,
        "filePath": "EmployeeApp\\internal\\routes\\routes.go"
      }
    ]
  },
  {
    "name": "pdf.ConvertHTMLToImage",
    "line": 59,
    "filePath": "gopdfsuit\\internal\\pdf\\pdf.go",
    "called": [
      {
        "name": "gochromedp.ConvertHTMLToImage",
        "line": 201,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      },
      {
        "name": "gochromedp.ConvertURLToImage",
        "line": 268,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      }
    ]
  },
  {
    "name": "pdf.ConvertHTMLToPDF",
    "line": 21,
    "filePath": "gopdfsuit\\internal\\pdf\\pdf.go",
    "called": [
      {
        "name": "gochromedp.ConvertHTMLToPDF",
        "line": 52,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      },
      {
        "name": "gochromedp.ConvertURLToPDF",
        "line": 128,
        "filePath": "gochromedp\\pkg\\gochromedp\\chrometopdf.go"
      }
    ]
  },
  {
    "name": "routes.SetupRouter",
    "line": 10,
    "filePath": "EmployeeApp\\internal\\routes\\routes.go",
    "called": [
      {
        "name": "middleware.CORS",
        "line": 5,
        "filePath": "EmployeeApp\\internal\\middleware\\cors.go"
      },
      {
        "name": "middleware.Logger",
        "line": 10,
        "filePath": "EmployeeApp\\internal\\middleware\\logger.go"
      },
      {
        "name": "handlers.NewEmployeeHandler",
        "line": 12,
        "filePath": "EmployeeApp\\internal\\handlers\\employee.go"
      }
    ]
  }
];

function App() {
  const [functionData, setFunctionData] = useState([]);
  const [selectedNode, setSelectedNode] = useState(null);

  useEffect(() => {
    // In a real app, you would fetch this from your functionmap.json
    setFunctionData(sampleData);
  }, []);

  return (
    <div className="App">
      <header className="app-header">
        <h1>Function Mind Map</h1>
        <p>Visualize your Go application's function call hierarchy</p>
      </header>
      
      <main className="app-main">
        <MindMap 
          data={functionData} 
          selectedNode={selectedNode}
          onNodeSelect={setSelectedNode}
        />
      </main>
      
      {selectedNode && (
        <div className="node-details">
          <h3>Function Details</h3>
          <p><strong>Name:</strong> {selectedNode.name}</p>
          <p><strong>Line:</strong> {selectedNode.line}</p>
          <p><strong>File:</strong> {selectedNode.filePath}</p>
          {selectedNode.called && selectedNode.called.length > 0 && (
            <div>
              <strong>Calls:</strong>
              <ul>
                {selectedNode.called.map((fn, index) => (
                  <li key={index}>{fn.name} (line {fn.line})</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default App;