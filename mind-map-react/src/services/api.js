class ApiService {
  constructor(baseUrl = 'http://localhost:8080/api') {
    this.baseUrl = baseUrl;
  }

  async get(endpoint) {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  async getFunctions(page = 1, pageSize = 50) {
    return this.get(`/functions?page=${page}&pageSize=${pageSize}`);
  }

  async searchFunctions(query) {
    return this.get(`/functions/search?q=${encodeURIComponent(query)}`);
  }

  async getFunction(functionName) {
    return this.get(`/functions/${encodeURIComponent(functionName)}`);
  }

  async getFunctionWithDependencies(functionName) {
    return this.get(`/functions/${encodeURIComponent(functionName)}/dependencies`);
  }
}

const apiService = new ApiService();
export default apiService;