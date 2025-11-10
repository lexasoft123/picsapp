import React, { useState, useEffect, useRef } from 'react';
import './Presentation.css';

function Presentation() {
  const [pictures, setPictures] = useState([]);
  const [loading, setLoading] = useState(true);
  const wsRef = useRef(null);
  const prevPositionsRef = useRef(new Map());
  const [swappingIds, setSwappingIds] = useState(new Set());
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const isInitialLoadRef = useRef(true);

  useEffect(() => {
    let ws = null;
    let isMounted = true;
    let reconnectTimeout = null;

    // Initial fetch
    fetch('/api/presentation')
      .then((res) => {
        if (!res.ok) {
          throw new Error('Failed to fetch presentation');
        }
        return res.json();
      })
      .then((data) => {
        if (isMounted) {
          const newPictures = Array.isArray(data) ? data : [];
          // Initialize previous positions
          const positions = new Map();
          newPictures.forEach((pic, index) => {
            positions.set(pic.id, index);
          });
          prevPositionsRef.current = positions;
          setPictures(newPictures);
          setLoading(false);
          isInitialLoadRef.current = false;
          setIsInitialLoad(false);
        }
      })
      .catch((err) => {
        console.error('Error fetching presentation:', err);
        if (isMounted) {
          setPictures([]);
          setLoading(false);
        }
      });

    // WebSocket connection
    // In development, connect directly to the Go server on port 8080
    // In production, use the same host
    const isDev = window.location.hostname === 'localhost' && window.location.port === '3000';
    const wsHost = isDev ? 'localhost:8080' : window.location.host;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${wsHost}/ws`;

    const connectWebSocket = () => {
      if (!isMounted) return;

      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected');
        if (reconnectTimeout) {
          clearTimeout(reconnectTimeout);
          reconnectTimeout = null;
        }
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (isMounted) {
            const newPictures = Array.isArray(data) ? data : [];
            
            // Detect position changes
            const newPositions = new Map();
            const changedIds = new Set();
            
            newPictures.forEach((pic, index) => {
              newPositions.set(pic.id, index);
              const prevIndex = prevPositionsRef.current.get(pic.id);
              if (prevIndex !== undefined && prevIndex !== index) {
                changedIds.add(pic.id);
              }
            });
            
            // Also check if any picture moved from a different position
            prevPositionsRef.current.forEach((prevIndex, id) => {
              const newIndex = newPositions.get(id);
              if (newIndex !== undefined && newIndex !== prevIndex) {
                changedIds.add(id);
              }
            });
            
            // Update swapping state
            if (changedIds.size > 0 && !isInitialLoadRef.current) {
              setSwappingIds(new Set(changedIds));
              // Clear swapping state after animation
              setTimeout(() => {
                if (isMounted) {
                  setSwappingIds(new Set());
                }
              }, 600);
            }
            
            // Update previous positions
            prevPositionsRef.current = newPositions;
            setPictures(newPictures);
            if (isInitialLoadRef.current) {
              isInitialLoadRef.current = false;
              setIsInitialLoad(false);
            }
          }
        } catch (err) {
          console.error('Error parsing WebSocket message:', err);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      ws.onclose = (event) => {
        if (!isMounted) return;

        if (event.wasClean) {
          console.log('WebSocket disconnected cleanly');
        } else {
          console.log('WebSocket connection lost, attempting to reconnect...');
          // Attempt to reconnect after a delay
          reconnectTimeout = setTimeout(() => {
            if (isMounted) {
              connectWebSocket();
            }
          }, 3000);
        }
      };

      wsRef.current = ws;
    };

    connectWebSocket();

    return () => {
      isMounted = false;
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
      if (wsRef.current) {
        if (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING) {
          wsRef.current.close();
        }
      }
    };
  }, []);

  if (loading) {
    return (
      <div className="presentation-page">
        <div className="loading">Loading presentation...</div>
      </div>
    );
  }

  const safePictures = pictures || [];

  return (
    <div className="presentation-page">
      <div className="container">
        <h1 className="page-title">Top Pictures by Likes</h1>
        <div className="presentation-grid">
          {safePictures.length === 0 ? (
            <div className="empty-state">
              <div className="empty-icon">🖼️</div>
              <div className="empty-text">No pictures yet.</div>
            </div>
          ) : (
            safePictures.map((picture, index) => {
              const isSwapping = swappingIds.has(picture.id);
              return (
              <div
                key={picture.id}
                className={`presentation-card ${isSwapping ? 'swapping' : ''} ${isInitialLoad ? 'initial' : ''}`}
                style={{ 
                  animationDelay: isInitialLoad ? `${index * 0.1}s` : '0s'
                }}
              >
                <div className="card-rank">#{index + 1}</div>
                <img
                  src={picture.url}
                  alt={picture.filename}
                  className="presentation-image"
                />
                <div className="card-info">
                  <div className="card-likes">
                    <span className="likes-icon">❤️</span>
                    <span className="likes-count">{picture.likes}</span>
                  </div>
                </div>
              </div>
            );
            })
          )}
        </div>
      </div>
    </div>
  );
}

export default Presentation;

