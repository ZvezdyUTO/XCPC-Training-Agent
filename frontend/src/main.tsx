import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { App } from "./app/App";
import { AuthProvider } from "./features/auth/AuthContext";
import { AgentRunProvider } from "./features/agent/AgentRunContext";
import "./shared/styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <AgentRunProvider>
          <App />
        </AgentRunProvider>
      </AuthProvider>
    </BrowserRouter>
  </React.StrictMode>,
);
