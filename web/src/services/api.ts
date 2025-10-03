import axios from 'axios';

// Use environment variable if available, otherwise localhost
const API_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/$/, '');
console.log('API_URL:', API_URL); // Add this line
// Create axios instance with base configuration
export const api = axios.create({
    baseURL: `${API_URL}/api`,
    withCredentials: false, // Important for CORS
    headers: {
        'Content-Type': 'application/json',
    }
});

// Export the base URL for other uses (like S3 uploads)
export const API_BASE_URL = API_URL;