import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Server, Users, Activity, Clock } from 'lucide-react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';

export default function Home() {
  const [stats, setStats] = useState(null);
  const [status, setStatus] = useState(null);
  const [commandStats, setCommandStats] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statusRes, cmdsRes] = await Promise.all([
          axios.get('http://localhost:8080/api/status'),
          axios.get('http://localhost:8080/api/stats/commands')
        ]);
        setStatus(statusRes.data);
        setCommandStats(cmdsRes.data || []);
      } catch (err) {
        console.error("Failed to fetch dashboard data:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();

    // WebSocket connection for real-time stats
    const ws = new WebSocket('ws://localhost:8080/ws');

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setStats(data);
      } catch (e) {
        console.error("Error parsing websocket message:", e);
      }
    };

    ws.onclose = () => {
      console.log("WebSocket connection closed.");
    };

    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
    };

    return () => {
      ws.close();
    };
  }, []);

  if (loading && !stats) {
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

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {status && (
          <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 shadow-sm flex flex-col justify-center">
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
            <p className="mt-4 text-sm text-gray-400">
              Stats are updating in real-time via WebSocket.
            </p>
          </div>
        )}

        <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 shadow-sm">
          <h3 className="text-lg font-medium text-white mb-4">Command Usage</h3>
          <div className="h-64">
            {commandStats.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={commandStats} layout="vertical" margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" horizontal={false} />
                  <XAxis type="number" stroke="#9CA3AF" />
                  <YAxis dataKey="name" type="category" stroke="#9CA3AF" width={100} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                    itemStyle={{ color: '#F3F4F6' }}
                  />
                  <Bar dataKey="count" fill="#8B5CF6" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                No command usage data available.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
