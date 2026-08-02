import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './hooks/useAuth';
import LandingPage from './pages/LandingPage';
import LoginPage from './pages/LoginPage';
import RepoListPage from './pages/RepoListPage';
import RepoDetailPage from './pages/RepoDetailPage';
import AccountPage from './pages/AccountPage';

function App() {
  const isAuthenticated = useAuth();

  if (isAuthenticated === null) {
    return <div className="min-h-screen flex items-center justify-center text-gray-500">Loading...</div>;
  }

  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/login" element={isAuthenticated ? <Navigate to="/dashboard" replace /> : <LoginPage />} />
      <Route path="/dashboard" element={isAuthenticated ? <RepoListPage /> : <Navigate to="/login" replace />} />
      <Route path="/dashboard/repos/:id" element={isAuthenticated ? <RepoDetailPage /> : <Navigate to="/login" replace />} />
      <Route path="/account" element={isAuthenticated ? <AccountPage /> : <Navigate to="/login" replace />} />
    </Routes>
  );
}

export default App;