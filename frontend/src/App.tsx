import { useEffect, useState } from 'react';
import { Navigate, Outlet, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import './App.css';
import { fetchWithAuth, getToken, setUnauthorizedHandler } from './lib/api';
import { LoginPage } from './legacy-pages/LoginPage';
import { RegisterPage } from './legacy-pages/RegisterPage';
import { AdminPage } from './legacy-pages/AdminPage';
import { ChatLayout } from './legacy-pages/ChatLayout';
import { SettingsPage } from './legacy-pages/SettingsPage';
import { SearchPage } from './legacy-pages/SearchPage';
import { BookmarksPage } from './legacy-pages/BookmarksPage';
import { VerifyEmailPage } from './legacy-pages/VerifyEmailPage';
import { ResetPasswordPage } from './legacy-pages/ResetPasswordPage';
import { ForgotPasswordPage } from './legacy-pages/ForgotPasswordPage';

function RequireAuth() {
  const location = useLocation();
  const [status, setStatus] = useState<'checking' | 'authed' | 'unauth'>(() =>
    getToken() ? 'checking' : 'unauth',
  );

  useEffect(() => {
    if (!getToken()) return;
    let alive = true;
    void (async () => {
      const res = await fetchWithAuth('/me');
      if (!alive) return;
      setStatus(res.ok ? 'authed' : 'unauth');
    })();
    return () => {
      alive = false;
    };
  }, []);

  if (status === 'checking') {
    return <div className="chat-empty">認証状態を確認中...</div>;
  }
  if (status !== 'authed') {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <Outlet />;
}

function HomeRedirect() {
  return <Navigate to={getToken() ? '/rooms' : '/login'} replace />;
}

function App() {
  const navigate = useNavigate();
  useEffect(() => {
    setUnauthorizedHandler(() => {
      navigate('/login', { replace: true });
    });
    return () => setUnauthorizedHandler(null);
  }, [navigate]);

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route element={<RequireAuth />}>
        <Route path="/rooms/*" element={<ChatLayout />} />
        <Route path="/admin" element={<AdminPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/bookmarks" element={<BookmarksPage />} />
      </Route>
      <Route path="/" element={<HomeRedirect />} />
      <Route path="*" element={<HomeRedirect />} />
    </Routes>
  );
}

export default App;
