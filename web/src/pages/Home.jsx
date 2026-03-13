import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Server, Users, Activity, Clock } from 'lucide-react';

export default function Home() {
  const [stats, setStats] = useState(null);
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statsRes, statusRes] = await Promise.all([
          axios.get('http://localhost:8080/api/stats'),
          axios.get('http://localhost:8080/api/status')
        ]);
        setStats(statsRes.data);
        setStatus(statusRes.data);
      } catch (err) {
        console.error("Failed to fetch dashboard data:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
      </div>
    );
  }

  const cards = [
    { name: 'Servers', value: stats?.guilds || 0, icon: <Server className="h-6 w-6 text-blue-400" />, color: 'bg-blue-900/50' },
    { name: 'Users', value: stats?.users || 0, icon: <Users className="h-6 w-6 text-green-400" />, color: 'bg-green-900/50' },
    { name: 'Commands Run', value: stats?.commands_run || 0, icon: <Activity className="h-6 w-6 text-purple-400" />, color: 'bg-purple-900/50' },
    { name: 'Uptime', value: stats?.uptime?.split('.')[0] || '0s', icon: <Clock className="h-6 w-6 text-orange-400" />, color: 'bg-orange-900/50' },
  ];

  return (
    <div className="space-y-6 text-gray-100">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {cards.map((card) => (
          <div key={card.name} className="bg-gray-800 rounded-xl p-6 border border-gray-700 shadow-sm flex items-center">
            <div className={`p-4 rounded-lg mr-4 ${card.color}`}>
              {card.icon}
            </div>
            <div>
              <p className="text-sm font-medium text-gray-400 mb-1">{card.name}</p>
              <h3 className="text-2xl font-bold text-white">{card.value}</h3>
            </div>
          </div>
        ))}
      </div>

      {status && (
        <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 shadow-sm">
          <h3 className="text-lg font-medium text-white mb-4">Bot Status</h3>
          <div className="flex items-center">
            <span className="flex h-3 w-3 relative mr-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
            </span>
            <span className="text-gray-300">
              System is <strong className="text-green-400 font-medium">Online</strong> and processing Discord events.
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
