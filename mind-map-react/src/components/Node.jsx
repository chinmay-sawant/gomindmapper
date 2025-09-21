import React from 'react';

const Node = ({ 
  node, 
  x, 
  y, 
  width = 200,
  isExpanded, 
  hasChildren, 
  isSelected, 
  onToggle, 
  onSelect, 
  level 
}) => {
  const getNodeColor = (level) => {
    const colors = [
      '#4f46e5', // indigo
      '#7c3aed', // violet  
      '#db2777', // pink
      '#dc2626', // red
      '#ea580c', // orange
      '#16a34a', // green
    ];
    return colors[level % colors.length];
  };

  const getFunctionType = (name) => {
    if (name.includes('main')) return 'main';
    if (name.includes('Handler')) return 'handler';
    if (name.includes('middleware') || name.includes('CORS') || name.includes('Logger')) return 'middleware';
    if (name.includes('config') || name.includes('Load')) return 'config';
    if (name.includes('routes') || name.includes('Router')) return 'router';
    return 'function';
  };

  const functionType = getFunctionType(node.name);
  const nodeColor = getNodeColor(level);
  
  // Create a more descriptive display name
  const getDisplayName = (node) => {
    const funcName = node.name.split('.').pop();
    if (node.name.endsWith('.main') && node.filePath) {
      // Derive context folder name just before main.go
      const parts = node.filePath.split(/\\|\//);
      let ctx = '';
      if (parts.length >= 2) {
        // choose the directory containing main.go (previous segment)
        ctx = parts[parts.length - 2];
      }
      if (!ctx || ctx === 'cmd') {
        // fallback: pick first non-empty segment
        ctx = parts.find(p => p && !p.endsWith('.go') && p !== 'cmd') || 'root';
      }
      return `${funcName} (${ctx})`;
    }
    return funcName;
  };
  
  const displayName = getDisplayName(node);
  
  return (
    <g
      className={`node ${functionType} ${isSelected ? 'selected' : ''}`}
      transform={`translate(${x}, ${y})`}
      onClick={(e) => {
        e.stopPropagation();
        onSelect();
      }}
      title={`${node.name}\n${node.filePath}:${node.line}`}
    >
      {/* Node background */}
      <rect
        x={0}
        y={-20}
        width={width}
        height={40}
        rx={8}
        className="node-bg"
        fill={isSelected ? nodeColor : '#2d2d2d'}
        stroke={nodeColor}
        strokeWidth={isSelected ? 3 : 2}
        filter={isSelected ? "url(#glow)" : ""}
      />
      
      {/* Function type indicator */}
      <circle
        cx={12}
        cy={0}
        r={4}
        fill={nodeColor}
        opacity={0.8}
      />
      
      {/* Node text */}
      <text
        x={25}
        y={6}
        textAnchor="start"
        className="node-text"
        fill={isSelected ? '#ffffff' : '#e5e5e5'}
        fontSize="14"
        fontWeight={isSelected ? '600' : '500'}
      >
        {displayName}
      </text>
      
      {/* Expand/collapse button */}
      {hasChildren && (
        <g
          className="expand-button"
          onClick={(e) => {
            e.stopPropagation();
            onToggle();
          }}
        >
          <circle
            cx={width + 12}
            cy={0}
            r={12}
            fill={nodeColor}
            stroke="#ffffff"
            strokeWidth={1}
            className="expand-circle"
          />
          <text
            x={width + 12}
            y={4}
            textAnchor="middle"
            fill="#ffffff"
            fontSize="14"
            fontWeight="bold"
            className="expand-icon"
          >
            {isExpanded ? '<' : '>'}
          </text>
        </g>
      )}
      
      {/* Children count indicator */}
      {hasChildren && (
        <text
          x={width / 2}
          y={-28}
          textAnchor="middle"
          className="children-count"
          fill={nodeColor}
          fontSize="10"
          fontWeight="500"
        >
          {node.children.length} call{node.children.length !== 1 ? 's' : ''}
        </text>
      )}
    </g>
  );
};

export default Node;