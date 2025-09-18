import React, { useState, useCallback, useEffect } from 'react';
import './SearchComponent.css';

const SearchComponent = ({ onFunctionSelect, onClearCanvas }) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [showResults, setShowResults] = useState(false);

  const debounce = (func, delay) => {
    let debounceTimer;
    return function (...args) {
      const context = this;
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => func.apply(context, args), delay);
    };
  };

  const performSearch = useCallback(async (query) => {
    if (!query.trim()) {
      setSearchResults([]);
      setShowResults(false);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`http://localhost:8080/api/functions/search?q=${encodeURIComponent(query)}`);
      if (!response.ok) {
        throw new Error(`Search failed: ${response.status}`);
      }
      const data = await response.json();
      setSearchResults(data.functions || []);
      setShowResults(true);
    } catch (err) {
      console.error('Search failed:', err);
      setError('Search failed. Please ensure the server is running.');
      setSearchResults([]);
      setShowResults(false);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const debouncedSearch = useCallback(
    debounce((query) => performSearch(query), 300), 
    [performSearch]
  );

  useEffect(() => {
    debouncedSearch(searchQuery);
  }, [searchQuery, debouncedSearch]);

  const handleInputChange = (e) => {
    setSearchQuery(e.target.value);
  };

  const handleFunctionSelect = async (functionItem) => {
    try {
      // Clear canvas first
      onClearCanvas();
      
      // Fetch function with its dependencies
      const response = await fetch(
        `http://localhost:8080/api/functions/${encodeURIComponent(functionItem.name)}/dependencies`
      );
      
      if (!response.ok) {
        throw new Error(`Failed to fetch function dependencies: ${response.status}`);
      }
      
      const functionData = await response.json();
      onFunctionSelect(functionData);
      
      // Clear search
      setSearchQuery('');
      setShowResults(false);
      setSearchResults([]);
    } catch (err) {
      console.error('Failed to load function:', err);
      setError('Failed to load function. Please try again.');
    }
  };

  const handleClearSearch = () => {
    setSearchQuery('');
    setSearchResults([]);
    setShowResults(false);
    setError(null);
  };

  return (
    <div className="search-component">
      <div className="search-input-container">
        <input
          type="text"
          value={searchQuery}
          onChange={handleInputChange}
          placeholder="Search for functions..."
          className="search-input"
        />
        {searchQuery && (
          <button 
            onClick={handleClearSearch}
            className="clear-search-btn"
            aria-label="Clear search"
          >
            ×
          </button>
        )}
      </div>
      
      {isLoading && (
        <div className="search-loading">
          Searching...
        </div>
      )}
      
      {error && (
        <div className="search-error">
          {error}
        </div>
      )}
      
      {showResults && searchResults.length > 0 && (
        <div className="search-results">
          <div className="search-results-header">
            Found {searchResults.length} function{searchResults.length !== 1 ? 's' : ''}
          </div>
          <ul className="search-results-list">
            {searchResults.map((result, index) => (
              <li 
                key={`${result.name}-${index}`}
                className="search-result-item"
                onClick={() => handleFunctionSelect(result)}
              >
                <div className="function-name">{result.name}</div>
                <div className="function-details">
                  <span className="file-path">{result.filePath}</span>
                  <span className="line-number">Line {result.line}</span>
                </div>
                {result.called && result.called.length > 0 && (
                  <div className="function-calls">
                    Calls {result.called.length} function{result.called.length !== 1 ? 's' : ''}
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
      
      {showResults && searchResults.length === 0 && !isLoading && searchQuery.trim() && (
        <div className="search-no-results">
          No functions found for "{searchQuery}"
        </div>
      )}
    </div>
  );
};

export default SearchComponent;