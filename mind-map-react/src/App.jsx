import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Routes, Route } from 'react-router-dom';
import Navbar from './components/Navbar';
import './components/Navbar.css';
import Overview from './components/Overview';
import MindMap from './components/MindMap';
import './App.css';

// Default data using current functionmap.json
const defaultEmployeeAppData = [
  {
    "name": "handlers.RegisterRoutes",
    "line": 77,
    "filePath": "gopdfsuit\\internal\\handlers\\handlers.go",
    "called": [
      {
        "name": "handlers.handleGenerateTemplatePDF",
        "line": 157,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      },
      {
        "name": "handlers.handleFillPDF",
        "line": 168,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      },
      {
        "name": "handlers.handleMergePDFs",
        "line": 222,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      },
      {
        "name": "handlers.handleGetTemplateData",
        "line": 119,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      },
      {
        "name": "handlers.handlehtmlToPDF",
        "line": 266,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      },
      {
        "name": "handlers.handlehtmlToImage",
        "line": 308,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      },
      {
        "name": "handlers.handleSPA",
        "line": 103,
        "filePath": "gopdfsuit\\internal\\handlers\\handlers.go"
      }
    ]
  },
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
  }
];

function App() {
  return (
    <Routes>
      <Route path="/" element={<Overview />} />
      <Route path="/view/*" element={<MindMapApp />} />
    </Routes>
  );
}

function MindMapApp() {
  const [functionData, setFunctionData] = useState(defaultEmployeeAppData);
  const [selectedNode, setSelectedNode] = useState(null);
  const [fileName, setFileName] = useState('EmployeeApp (Default)');
  const [dragActive, setDragActive] = useState(false);
  // eslint-disable-next-line no-unused-vars
  const [dragCounter, setDragCounter] = useState(0);
  // Pagination state when using backend or local file
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(5);
  const [totalRoots, setTotalRoots] = useState(0);
  const [useServer, setUseServer] = useState(false);
  const [loading, setLoading] = useState(false);
  const [serverError, setServerError] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);
  // Local file data management
  const [fullLocalData, setFullLocalData] = useState(null);
  const [useLocalPagination, setUseLocalPagination] = useState(false);
  const [localSearchResults, setLocalSearchResults] = useState([]);
  const appRef = useRef(null);
  const searchTimeoutRef = useRef(null);
  const searchInputRef = useRef(null);
  const isTypingRef = useRef(false);

  // Get root nodes from data (functions that aren't called by others)
  const getRootNodes = useCallback((data) => {
    const getUniqueKey = (fn) => `${fn.name}@${fn.filePath}`;
    
    return data.filter(fn => {
      const fnKey = getUniqueKey(fn);
      return !data.some(f => 
        f.called && f.called.some(c => getUniqueKey(c) === fnKey)
      );
    });
  }, []);

  // Client-side pagination for local data
  const paginateLocalData = useCallback((data, p, ps, query = '') => {
    let filteredData = data;
    
    if (query && query.trim() !== '') {
      // Search in function names, file paths, and called function names
      const searchTerm = query.trim().toLowerCase();
      const searchResults = data.filter(fn => {
        // Search in function name
        if (fn.name && fn.name.toLowerCase().includes(searchTerm)) return true;
        
        // Search in file path
        if (fn.filePath && fn.filePath.toLowerCase().includes(searchTerm)) return true;
        
        // Search in called function names
        if (fn.called && fn.called.some(called => 
          called.name && called.name.toLowerCase().includes(searchTerm)
        )) return true;
        
        return false;
      });
      
      setLocalSearchResults(searchResults);
      filteredData = searchResults;
    } else {
      setLocalSearchResults([]);
    }

    // Get root nodes from filtered data
    const roots = getRootNodes(filteredData);
    const totalRoots = roots.length;
    
    // Calculate pagination
    const startIndex = (p - 1) * ps;
    const endIndex = startIndex + ps;
    const paginatedRoots = roots.slice(startIndex, endIndex);
    
    // Build complete data with dependencies for paginated roots
    const buildCompleteData = (rootNodes, allData) => {
      const result = [];
      const processed = new Set();
      
      const processNode = (node) => {
        const key = `${node.name}@${node.filePath}`;
        if (processed.has(key)) return;
        processed.add(key);
        
        result.push(node);
        
        // Add all called functions
        if (node.called) {
          node.called.forEach(calledFn => {
            const fullCalledFn = allData.find(f => 
              f.name === calledFn.name && f.filePath === calledFn.filePath
            );
            if (fullCalledFn) {
              processNode(fullCalledFn);
            }
          });
        }
      };
      
      rootNodes.forEach(processNode);
      return result;
    };
    
    const completeData = buildCompleteData(paginatedRoots, data);
    
    return {
      data: completeData,
      totalRoots,
      page: p,
      pageSize: ps
    };
  }, [getRootNodes]);

  // Fetch paginated data from backend server
  const fetchPage = useCallback(async (p = page, ps = pageSize, query = '') => {
    if (!useServer) {
      // Handle local pagination
      if (useLocalPagination && fullLocalData) {
        setLoading(true);
        try {
          const result = paginateLocalData(fullLocalData, p, ps, query);
          setFunctionData(result.data);
          setTotalRoots(result.totalRoots);
          setPage(result.page);
          setPageSize(result.pageSize);
          
          const rootsCount = getRootNodes(fullLocalData).length;
          const label = query 
            ? `Local Search: "${query}" (${result.totalRoots} matches, page ${result.page})` 
            : `Local File (${rootsCount} total roots, page ${result.page})`;
          setFileName(label);
          setSelectedNode(null);
        } finally {
          setLoading(false);
        }
      }
      return;
    }
    
    setLoading(true);
    setServerError('');
    try {
      const url = query 
        ? `${window.location.origin}/api/search?q=${encodeURIComponent(query)}&page=${p}&pageSize=${ps}`
        : `${window.location.origin}/api/relations?page=${p}&pageSize=${ps}`;
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      // json.data is the closure; we use it directly for mind map
      setFunctionData(json.data || []);
      if (query) {
        // For search results, use totalResults for pagination
        setTotalRoots(json.totalResults || 0);
      } else {
        // For normal pagination, use totalRoots
        setTotalRoots(json.totalRoots || 0);
      }
      setPage(json.page || p);
      setPageSize(json.pageSize || ps);
      const label = query 
        ? `Search: "${query}" (${json.totalResults || 0} matches, page ${json.page})` 
        : `Server Roots Page ${json.page}`;
      setFileName(label);
      setSelectedNode(null);
    } catch (e) {
      setServerError(e.message);
    } finally {
      setLoading(false);
    }
  }, [useServer, useLocalPagination, fullLocalData, paginateLocalData, getRootNodes]);

  // Auto fetch when toggling useServer or page changes (but not when there's an active search)
  useEffect(() => {
    if ((useServer || useLocalPagination) && (!searchQuery || searchQuery.trim() === '')) {
      fetchPage(page, pageSize);
    }
  }, [useServer, useLocalPagination, page, pageSize, fetchPage, searchQuery]);

  // Clear search when switching off server mode or local pagination
  useEffect(() => {
    if (!useServer && !useLocalPagination) {
      setSearchQuery('');
      setSearchResults([]);
      setLocalSearchResults([]);
    }
  }, [useServer, useLocalPagination]);

  // Search function with debouncing
  const handleSearch = useCallback(async (query, immediate = false, searchPage = 1, searchPageSize = pageSize) => {
    if (!useServer && !useLocalPagination) return;
    
    // Clear existing timeout
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }
    
    const performSearch = async () => {
      // Store the currently focused element before search
      const activeElement = document.activeElement;
      const wasFocusedOnSearchInput = activeElement === searchInputRef.current;
      
      setIsSearching(true);
      if (query.trim() === '') {
        // If search is cleared, go back to paginated view, use current pageSize
        await fetchPage(1, searchPageSize);
        setSearchResults([]);
      } else {
        // Perform search with pagination, use current pageSize
        await fetchPage(searchPage, searchPageSize, query.trim());
      }
      setIsSearching(false);
      
      // Restore focus to search input if it was focused before
      if (wasFocusedOnSearchInput && searchInputRef.current) {
        // Use a small timeout to ensure the DOM has updated
        setTimeout(() => {
          if (searchInputRef.current && !isTypingRef.current) {
            searchInputRef.current.focus();
          }
        }, 10);
      }
    };
    
    if (immediate) {
      await performSearch();
    } else {
      // Increase debounce time to 1500ms to give more time for typing
      searchTimeoutRef.current = setTimeout(performSearch, 1500);
    }
  }, [useServer, useLocalPagination, fetchPage]);

  // Handle search input changes with debouncing
  const handleSearchInputChange = useCallback((newQuery) => {
    setSearchQuery(newQuery);
    // Reset to page 1 when search query changes
    setPage(1);
    handleSearch(newQuery, false, 1);
  }, [handleSearch]);

  const handleFileUpload = useCallback((file) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const jsonData = JSON.parse(e.target.result);
        
        // Check if the data is large enough to warrant pagination
        const rootNodes = getRootNodes(jsonData);
        const shouldUsePagination = rootNodes.length > 10; // Enable pagination if more than 10 root nodes
        
        if (shouldUsePagination) {
          // Store full data and enable local pagination
          setFullLocalData(jsonData);
          setUseLocalPagination(true);
          setUseServer(false); // Disable server mode
          setPage(1); // Reset to first page
          setSearchQuery(''); // Clear any existing search
          
          // Paginate the data immediately
          const result = paginateLocalData(jsonData, 1, pageSize);
          setFunctionData(result.data);
          setTotalRoots(result.totalRoots);
          
          setFileName(`${file.name} (${rootNodes.length} total roots, paginated)`);
          alert(`Large dataset detected! Loaded ${rootNodes.length} root functions with pagination enabled. Use the pagination controls and search to navigate through the data.`);
        } else {
          // Small dataset, load normally
          setFullLocalData(null);
          setUseLocalPagination(false);
          setFunctionData(jsonData);
          setTotalRoots(rootNodes.length);
          setFileName(file.name);
        }
        
        setSelectedNode(null); // Clear selection when new data loads
      } catch (error) {
        alert('Error parsing JSON file: ' + error.message);
      }
    };
    reader.readAsText(file);
  }, [getRootNodes, paginateLocalData, pageSize]);

  const handleDrop = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    setDragCounter(0);
    
    const files = [...e.dataTransfer.files];
    if (files && files[0]) {
      const file = files[0];
      if (file.type === 'application/json' || file.name.endsWith('.json')) {
        handleFileUpload(file);
      } else {
        alert('Please upload a JSON file');
      }
    }
  }, [handleFileUpload]);

  const handleDragEnter = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragCounter(prev => prev + 1);
    if (!dragActive) {
      setDragActive(true);
    }
  }, [dragActive]);

  const handleDragLeave = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragCounter(prev => {
      const newCounter = prev - 1;
      // Only hide overlay when counter reaches 0 (actually left the container)
      if (newCounter === 0) {
        // Add a small delay to prevent flickering
        setTimeout(() => {
          setDragActive(false);
        }, 100);
      }
      return newCounter;
    });
  }, []);

  const handleDragOver = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleFileSelect = useCallback((e) => {
    const file = e.target.files[0];
    if (file) {
      handleFileUpload(file);
    }
  }, [handleFileUpload]);

  // Set up drag event listeners with non-passive option to allow preventDefault
  useEffect(() => {
    const appElement = appRef.current;
    if (!appElement) return;

    const handleDropNonPassive = (e) => handleDrop(e);
    const handleDragEnterNonPassive = (e) => handleDragEnter(e);
    const handleDragLeaveNonPassive = (e) => handleDragLeave(e);
    const handleDragOverNonPassive = (e) => handleDragOver(e);

    // Add event listeners with passive: false to allow preventDefault
    appElement.addEventListener('drop', handleDropNonPassive, { passive: false });
    appElement.addEventListener('dragenter', handleDragEnterNonPassive, { passive: false });
    appElement.addEventListener('dragleave', handleDragLeaveNonPassive, { passive: false });
    appElement.addEventListener('dragover', handleDragOverNonPassive, { passive: false });

    // Cleanup function
    return () => {
      appElement.removeEventListener('drop', handleDropNonPassive);
      appElement.removeEventListener('dragenter', handleDragEnterNonPassive);
      appElement.removeEventListener('dragleave', handleDragLeaveNonPassive);
      appElement.removeEventListener('dragover', handleDragOverNonPassive);
      
      // Clear search timeout on cleanup
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, [handleDrop, handleDragEnter, handleDragLeave, handleDragOver]);

  return (
    <div className="App" ref={appRef}>
      <Navbar onReload={useServer ? () => fetch(`${window.location.origin}/api/reload`, {method:'POST'}).then(()=>fetchPage(1,pageSize)) : null} onDownload={useServer ? `${window.location.origin}/api/download` : null} />
      <header className="app-header">
        <h1>Function Mind Map</h1>
        <div className="header-content">
          <p>Visualize your Go application's function call hierarchy</p>
          <div className="file-input-section">
            <span className="current-file">Current: {fileName}</span>
            <label htmlFor="file-input" className="file-input-label">Choose JSON File</label>
            <input id="file-input" type="file" accept=".json,application/json" onChange={handleFileSelect} className="file-input" />
            <label className="server-toggle">
              <input
                type="checkbox"
                checked={useServer}
                onChange={(e) => {
                  setUseServer(e.target.checked);
                  if (!e.target.checked) {
                    // If we have local paginated data, go back to it
                    if (useLocalPagination && fullLocalData) {
                      const result = paginateLocalData(fullLocalData, 1, pageSize);
                      setFunctionData(result.data);
                      setTotalRoots(result.totalRoots);
                      setPage(1);
                      const rootNodes = getRootNodes(fullLocalData);
                      setFileName(`Local File (${rootNodes.length} total roots, paginated)`);
                    } else {
                      // No local data, go back to default
                      setUseLocalPagination(false);
                      setFullLocalData(null);
                      setFunctionData(defaultEmployeeAppData);
                      setFileName('EmployeeApp (Default)');
                      setTotalRoots(0);
                    }
                    setSelectedNode(null);
                    setSearchQuery('');
                    setSearchResults([]);
                    setLocalSearchResults([]);
                  }
                }}
              /> Use Live Server
            </label>
            {useLocalPagination && !useServer && (
              <span className="pagination-indicator">📄 Local File Pagination Active</span>
            )}
            {(useServer || useLocalPagination) && (
              <div className="server-controls">
                <div className="search-bar">
                  <input 
                    ref={searchInputRef}
                    type="text" 
                    placeholder="Search functions..." 
                    value={searchQuery} 
                    onChange={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      isTypingRef.current = true;
                      handleSearchInputChange(e.target.value);
                      // Clear typing flag after a short delay
                      setTimeout(() => {
                        isTypingRef.current = false;
                      }, 100);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        e.stopPropagation();
                        isTypingRef.current = false;
                        handleSearch(searchQuery, true, 1, pageSize);
                      }
                    }}
                    onFocus={(e) => {
                      e.stopPropagation();
                      isTypingRef.current = true;
                    }}
                    onBlur={(e) => {
                      e.stopPropagation();
                      // Add delay before clearing typing flag to handle quick refocus
                      setTimeout(() => {
                        isTypingRef.current = false;
                      }, 200);
                    }}
                    className="search-input"
                    disabled={loading || isSearching}
                    autoComplete="off"
                  />
                  <button 
                    onClick={() => handleSearch(searchQuery, true, 1, pageSize)} 
                    disabled={loading || isSearching}
                    className="search-btn"
                  >
                    {isSearching ? '...' : 'Search'}
                  </button>
                  {searchQuery && searchQuery.trim() !== '' && (
                    <button 
                      onClick={() => {
                        setSearchQuery('');
                        handleSearch('', true, 1, pageSize);
                      }} 
                      disabled={loading || isSearching}
                      className="clear-search-btn"
                    >
                      Clear
                    </button>
                  )}
                </div>
                <div className="pagination-bar">
                  <button 
                    className="pg-btn" 
                    disabled={loading || page<=1} 
                    onClick={() => {
                      const newPage = Math.max(1, page-1);
                      setPage(newPage);
                      if (searchQuery && searchQuery.trim() !== '') {
                        handleSearch(searchQuery, true, newPage, pageSize);
                      } else {
                        fetchPage(newPage, pageSize);
                      }
                    }}
                  >&lt;</button>
                  <div className="pg-status">
                    {searchQuery && searchQuery.trim() !== '' ? 'Search ' : ''}Page {page} / {Math.max(1, Math.ceil(totalRoots / pageSize) || 1)}
                  </div>
                  <button 
                    className="pg-btn" 
                    disabled={loading || page >= Math.ceil(totalRoots / pageSize)} 
                    onClick={() => {
                      const newPage = page + 1;
                      setPage(newPage);
                      if (searchQuery && searchQuery.trim() !== '') {
                        handleSearch(searchQuery, true, newPage, pageSize);
                      } else {
                        fetchPage(newPage, pageSize);
                      }
                    }}
                  >&gt;</button>
                  <select 
                    className="pg-select" 
                    disabled={loading} 
                    value={pageSize} 
                    onChange={async (e)=> { 
                      const newPageSize = parseInt(e.target.value,10);
                      setPageSize(newPageSize);
                      setPage(1);
                      if (searchQuery && searchQuery.trim() !== '') {
                        await handleSearch(searchQuery, true, 1, newPageSize);
                      } else {
                        await fetchPage(1, newPageSize);
                      }
                    }}
                  >
                    {[5,10,15,20,50].map(n => <option key={n} value={n}>{n}/page</option>)}
                  </select>
                  <button 
                    className="pg-refresh" 
                    disabled={loading} 
                    onClick={() => {
                      if (searchQuery && searchQuery.trim() !== '') {
                        handleSearch(searchQuery, true, page, pageSize);
                      } else {
                        fetchPage(page, pageSize);
                      }
                    }}
                  >
                    {loading ? '...' : 'Refresh'}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </header>
      
      {dragActive && (
        <div className="drag-overlay">
          <div className="drag-message">
            <h2>Drop JSON file here</h2>
            <p>Release to load function map data</p>
          </div>
        </div>
      )}
      
      <main className="app-main">
        {(useServer || useLocalPagination) && loading && <div className="loading-indicator">Loading...</div>}
        {useServer && serverError && <div className="error-indicator">Error: {serverError}</div>}
        <MindMap 
          data={functionData} 
          selectedNode={selectedNode}
          onNodeSelect={setSelectedNode}
        />
      </main>
      
      {selectedNode && (
        <div className="node-details">
          <div className="node-details-header">
            <h3>Function Details</h3>
            <button 
              className="close-btn"
              onClick={() => setSelectedNode(null)}
              aria-label="Close details"
            >
              ×
            </button>
          </div>
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