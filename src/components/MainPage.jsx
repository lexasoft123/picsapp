import React, { useState, useEffect, useRef } from 'react';
import Upload from './Upload';
import PictureGrid from './PictureGrid';
import './MainPage.css';

function MainPage() {
  const [pictures, setPictures] = useState([]);
  const [loading, setLoading] = useState(true);
  const [dragActive, setDragActive] = useState(false);
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef(null);

  const fetchPictures = async () => {
    try {
      const response = await fetch('/api/pictures');
      if (!response.ok) {
        throw new Error('Failed to fetch pictures');
      }
      const data = await response.json();
      setPictures(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Error fetching pictures:', error);
      setPictures([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPictures();
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
        fetchPictures();
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
      const response = await fetch(`/api/pictures/${id}/like`, {
        method: 'POST',
      });
      if (response.ok) {
        fetchPictures();
      }
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

