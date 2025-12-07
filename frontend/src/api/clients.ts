// src/api/client.ts
import axios from "axios";

const baseURL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:5000/api/v1";

export const apiClient = axios.create({
  baseURL,
  timeout: 10000,
});

// Optional: add interceptors later for auth, logging, etc.
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    // centralised error handling if you want
    return Promise.reject(error);
  }
);
