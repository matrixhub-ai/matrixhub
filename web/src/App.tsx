import { BrowserRouter, Routes, Route, useLocation } from 'react-router-dom';
import { HomePage } from './pages/HomePage';
import { RepoPage } from './pages/RepoPage';
import { BlobPage } from './pages/BlobPage';
import { QueuePage } from './pages/QueuePage';
import './App.css';

// Router component that determines whether to show RepoPage or BlobPage
function PathRouter() {
  const location = useLocation();
  const pathname = location.pathname;
  
  // Check if URL contains /blob/ to show BlobPage
  if (pathname.includes('/blob/')) {
    return <BlobPage />;
  }
  
  // Otherwise show RepoPage (handles /tree/ and direct repo paths)
  return <RepoPage />;
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/queue" element={<QueuePage />} />
        <Route path="/*" element={<PathRouter />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;

