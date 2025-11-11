import React, { useState, useEffect, useRef } from 'react';
import Upload from './Upload';
import PictureGrid from './PictureGrid';
import './MainPage.css';

function MainPage() {
  const [pictures, setPictures] = useState([]);
  const [loading, setLoading] = useState(true);
  const [dragActive, setDragActive] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadMessage, setUploadMessage] = useState('');
  const fileInputRef = useRef(null);
  const wsRef = useRef(null);

  const selectHomePictures = React.useCallback((list) => {
    if (!Array.isArray(list)) {
      return [];
    }
    const sorted = [...list].sort((a, b) => {
      const dateA = new Date(a.uploadedAt).getTime();
      const dateB = new Date(b.uploadedAt).getTime();
      return dateB - dateA;
    });
    return sorted.slice(0, 30);
  }, []);

  const fetchPictures = async () => {
    try {
      const response = await fetch('/api/pictures');
      if (!response.ok) {
        throw new Error('Failed to fetch pictures');
      }
      const data = await response.json();
      setPictures(selectHomePictures(data));
    } catch (error) {
      console.error('Error fetching pictures:', error);
      setPictures([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPictures();

    let isMounted = true;
    let reconnectTimeout = null;

    // WebSocket connection
    // In development, connect directly to the Go server on port 8080
    // In production, use the same host
    const isDev = window.location.hostname === 'localhost' && window.location.port === '3000';
    const wsHost = isDev ? 'localhost:8080' : window.location.host;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${wsHost}/ws`;

    const connectWebSocket = () => {
      if (!isMounted) return;

      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected');
        if (reconnectTimeout) {
          clearTimeout(reconnectTimeout);
          reconnectTimeout = null;
        }
        setUploadMessage((prev) => (prev ? prev : ''));
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (isMounted) {
            setPictures(selectHomePictures(data));
            setLoading(false);
            setUploadMessage('');
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
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

  const handleUpload = async (file) => {
    if (!file || !file.type.startsWith('image/')) {
      alert('Please select an image file');
      return;
    }

    setUploading(true);
    const formData = new FormData();
    formData.append('picture', file);

    try {
      const response = await fetch('/api/upload', {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        setUploadMessage('Image queued. Processing…');
      } else {
        alert('Upload failed. Please try again.');
      }
    } catch (error) {
      console.error('Upload error:', error);
      alert('Upload failed. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  const handleDrag = (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    }
  };

  const handleDragLeave = (e) => {
    e.preventDefault();
    e.stopPropagation();
    // Only deactivate if we're leaving the main page container
    if (!e.currentTarget.contains(e.relatedTarget)) {
      setDragActive(false);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleUpload(e.dataTransfer.files[0]);
    }
  };

  const handleFileInput = (e) => {
    if (e.target.files && e.target.files[0]) {
      handleUpload(e.target.files[0]);
    }
  };

  const handleLike = async (id) => {
    try {
      await fetch(`/api/pictures/${id}/like`, {
        method: 'POST',
      });
    } catch (error) {
      console.error('Error liking picture:', error);
    }
  };

  return (
    <div 
      className={`main-page ${dragActive ? 'drag-active' : ''}`}
      onDragEnter={handleDrag}
      onDragLeave={handleDragLeave}
      onDragOver={handleDrag}
      onDrop={handleDrop}
    >
      <div className="container">
        <Upload 
          fileInputRef={fileInputRef}
          onFileSelect={handleFileInput}
          uploading={uploading}
        />
        {uploadMessage && (
          <div className="upload-status">{uploadMessage}</div>
        )}
        {loading ? (
          <div className="loading">Loading pictures...</div>
        ) : (
          <PictureGrid pictures={pictures} onLike={handleLike} />
        )}
      </div>
    </div>
  );
}

export default MainPage;

