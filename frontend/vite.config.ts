import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const proxyTarget = process.env.VITE_API_PROXY_TARGET || "http://localhost:8888";

// 开发态通过代理把 /v1 请求转发到后端，避免额外修改后端 CORS。
export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/v1": {
        target: proxyTarget,
        changeOrigin: true,
      },
    },
  },
});
