import React from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { Home, Server, Users, Shield, Settings, Activity, Coins } from 'lucide-react';

export default function Layout() {
  const navItems = [
    { name: 'Dashboard', path: '/', icon: <Home className="w-5 h-5 mr-3" /> },
    { name: 'Guilds', path: '/guilds', icon: <Server className="w-5 h-5 mr-3" /> },
    { name: 'Users', path: '/users', icon: <Users className="w-5 h-5 mr-3" /> },
    { name: 'Economy', path: '/economy', icon: <Coins className="w-5 h-5 mr-3" /> },
    { name: 'Moderation', path: '/moderation', icon: <Shield className="w-5 h-5 mr-3" /> },
    { name: 'Metrics', path: '/metrics', icon: <Activity className="w-5 h-5 mr-3" /> },
    { name: 'Config', path: '/config', icon: <Settings className="w-5 h-5 mr-3" /> },
  ];

  return (
    <div className="flex h-screen bg-gray-900 text-gray-100 font-sans">
      {/* Sidebar */}
      <div className="w-64 bg-gray-800 border-r border-gray-700 flex flex-col">
        <div className="p-6 border-b border-gray-700 flex items-center">
          <div className="w-8 h-8 bg-indigo-500 rounded-lg flex items-center justify-center mr-3">
            <span className="text-white font-bold text-xl">J</span>
          </div>
          <h1 className="text-xl font-bold tracking-tight text-white">JulesCord</h1>
        </div>

        <nav className="flex-1 py-4 overflow-y-auto">
          <ul className="space-y-1 px-3">
            {navItems.map((item) => (
              <li key={item.name}>
                <NavLink
                  to={item.path}
                  className={({ isActive }) =>
                    `flex items-center px-3 py-2.5 rounded-md transition-colors ${
                      isActive
                        ? 'bg-indigo-600 text-white'
                        : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                    }`
                  }
                >
                  {item.icon}
                  {item.name}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>

        <div className="p-4 border-t border-gray-700 text-sm text-gray-400 text-center">
          JulesCord v1.0.0
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="h-16 bg-gray-800 border-b border-gray-700 flex items-center px-6 shadow-sm">
          <h2 className="text-lg font-medium text-white">Dashboard</h2>
        </header>

        <main className="flex-1 overflow-y-auto bg-gray-900 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
