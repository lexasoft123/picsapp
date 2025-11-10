import React from 'react';
import './Upload.css';

function Upload({ fileInputRef, onFileSelect, uploading }) {
  const handleClick = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  return (
    <div className="upload-section">
      <input
        ref={fileInputRef}
        type="file"
        id="file-input"
        accept="image/*"
        onChange={onFileSelect}
        className="file-input"
      />
      <div className="upload-area">
        <div className="upload-label" onClick={handleClick}>
          <div className="upload-icon">📷</div>
          <div className="upload-text">
            {uploading ? 'Uploading...' : 'Drag & drop anywhere or click to upload'}
          </div>
          {uploading && (
            <div className="upload-progress">
              <div className="upload-progress-bar"></div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default Upload;

