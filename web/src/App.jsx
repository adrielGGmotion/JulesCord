import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Home from './pages/Home';
import Guilds from './pages/Guilds';
import Users from './pages/Users';
import Moderation from './pages/Moderation';
import Metrics from './pages/Metrics';
import GuildSettings from './pages/GuildSettings';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="guilds" element={<Guilds />} />
          <Route path="users" element={<Users />} />
          <Route path="moderation" element={<Moderation />} />
          <Route path="metrics" element={<Metrics />} />
          <Route path="guilds/:id/settings" element={<GuildSettings />} />
          <Route path="*" element={
            <div className="flex items-center justify-center h-full text-gray-400">
              <div className="text-center">
                <h2 className="text-2xl font-bold mb-2">Coming Soon</h2>
                <p>This module is under development.</p>
              </div>
            </div>
          } />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
