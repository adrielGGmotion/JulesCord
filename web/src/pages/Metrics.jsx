import React, { useState, useEffect } from 'react';
import { apiClient } from '../api';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Legend } from 'recharts';

export default function Metrics() {
  const [metrics, setMetrics] = useState({ commands: [], dbLatency: [], errorRates: [] });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await apiClient.get('/api/dashboard-metrics');
        setMetrics(res.data);
      } catch (err) {
        console.error("Failed to fetch dashboard metrics:", err);
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

  return (
    <div className="space-y-6 text-gray-100">
      <h2 className="text-2xl font-bold mb-6 text-white">System Metrics</h2>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 shadow-sm">
          <h3 className="text-lg font-medium text-white mb-4">Command Executions</h3>
          <div className="h-64">
            {metrics.commands && metrics.commands.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={metrics.commands} layout="vertical" margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" horizontal={false} />
                  <XAxis type="number" stroke="#9CA3AF" />
                  <YAxis dataKey="command" type="category" stroke="#9CA3AF" width={100} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                    itemStyle={{ color: '#F3F4F6' }}
                  />
                  <Bar dataKey="executions" fill="#8B5CF6" radius={[0, 4, 4, 0]} name="Executions" />
                  <Bar dataKey="latency" fill="#10B981" radius={[0, 4, 4, 0]} name="Avg Latency (ms)" />
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                No command metrics available.
              </div>
            )}
          </div>
        </div>

        <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 shadow-sm">
          <h3 className="text-lg font-medium text-white mb-4">DB Query Latency (ms)</h3>
          <div className="h-64">
            {metrics.dbLatency && metrics.dbLatency.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={metrics.dbLatency} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
                  <XAxis dataKey="query" stroke="#9CA3AF" />
                  <YAxis type="number" stroke="#9CA3AF" />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                    itemStyle={{ color: '#F3F4F6' }}
                  />
                  <Legend />
                  <Bar dataKey="latency" fill="#3B82F6" radius={[4, 4, 0, 0]} name="Avg Latency (ms)" />
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                No database query metrics available.
              </div>
            )}
          </div>
        </div>

        <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 shadow-sm col-span-1 lg:col-span-2">
          <h3 className="text-lg font-medium text-white mb-4">System Error Rates</h3>
          <div className="h-64">
            {metrics.errorRates && metrics.errorRates.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={metrics.errorRates} layout="vertical" margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" horizontal={false} />
                  <XAxis type="number" stroke="#9CA3AF" />
                  <YAxis dataKey="type" type="category" stroke="#9CA3AF" width={150} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1F2937', borderColor: '#374151', color: '#F3F4F6' }}
                    itemStyle={{ color: '#F3F4F6' }}
                  />
                  <Bar dataKey="count" fill="#EF4444" radius={[0, 4, 4, 0]} name="Error Count" />
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-500">
                No error metrics available. System is healthy.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
