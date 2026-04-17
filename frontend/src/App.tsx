import { Routes, Route, Navigate } from 'react-router-dom';
import './App.css';
import { LoginPage } from './pages/LoginPage';
import { ChatLayout } from './pages/ChatLayout';

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/rooms/*" element={<ChatLayout />} />
      <Route path="*" element={<Navigate to="/rooms" replace />} />
    </Routes>
  );
}

export default App;
