import React, { useState, useEffect } from 'react';
import './ScreenshotSlideshow.css';

const ScreenshotSlideshow = () => {
  const [currentSlide, setCurrentSlide] = useState(0);
  
  // Using GoMindMapper screenshots - we'll create placeholder URLs since screenshots folder is empty
  // In a real scenario, these would be actual screenshot URLs from the repository
  const screenshots = [
    {
      url: 'https://raw.githubusercontent.com/chinmay-sawant/gomindmapper/refs/heads/master/screenshots/view.png', // The one existing screenshot in the repo
      title: 'Interactive Mind Map View',
      description: 'Visualize your Go function call hierarchy with an interactive, expandable mind map interface.'
    },
    // {
    //   url: 'https://raw.githubusercontent.com/chinmay-sawant/gomindmapper/refs/heads/master/screenshots/1.png',
    //   title: 'Overview Dashboard',
    //   description: 'Get started with GoMindMapper through a clean, intuitive overview of your project structure.'
    // },
    {
      url: 'https://raw.githubusercontent.com/chinmay-sawant/gomindmapper/refs/heads/master/screenshots/functiondetails.png',
      title: 'Function Details Panel',
      description: 'Click on any function node to view detailed information including file path, line numbers, and call relationships.'
    },
    {
      url: 'https://raw.githubusercontent.com/chinmay-sawant/gomindmapper/refs/heads/master/screenshots/liveserver.png',
      title: 'Live Server Integration',
      description: 'Connect to your live Go server for real-time function mapping and pagination through large codebases.'
    }
  ];

  const nextSlide = () => {
    setCurrentSlide((prev) => (prev + 1) % screenshots.length);
  };

  const prevSlide = () => {
    setCurrentSlide((prev) => (prev - 1 + screenshots.length) % screenshots.length);
  };

  const goToSlide = (index) => {
    setCurrentSlide(index);
  };

  // Auto-play functionality
  useEffect(() => {
    const interval = setInterval(() => {
      nextSlide();
    }, 5000); // Change slide every 5 seconds

    return () => clearInterval(interval);
  }, []);

  return (
    <section className="screenshot-slideshow">
      <div className="slideshow-container">
        <div className="slides-wrapper">
          {screenshots.map((screenshot, index) => (
            <div
              key={index}
              className={`slide ${index === currentSlide ? 'active' : ''}`}
            >
              <div className="slide-content">
                <div className="slide-image">
                  <img
                    src={screenshot.url}
                    alt={screenshot.title}
                    loading="lazy"
                  />
                </div>
                <div className="slide-info">
                  <h3>{screenshot.title}</h3>
                  <p>{screenshot.description}</p>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Navigation controls */}
        <button className="nav-btn prev-btn" onClick={prevSlide} aria-label="Previous slide">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z"/>
          </svg>
        </button>
        
        <button className="nav-btn next-btn" onClick={nextSlide} aria-label="Next slide">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z"/>
          </svg>
        </button>

        {/* Slide indicators */}
        <div className="slide-indicators">
          {screenshots.map((_, index) => (
            <button
              key={index}
              className={`indicator ${index === currentSlide ? 'active' : ''}`}
              onClick={() => goToSlide(index)}
              aria-label={`Go to slide ${index + 1}`}
            />
          ))}
        </div>
      </div>
    </section>
  );
};

export default ScreenshotSlideshow;