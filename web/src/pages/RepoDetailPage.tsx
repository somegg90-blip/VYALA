import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import DashboardLayout from '../components/DashboardLayout';
import type { TrendPoint, Finding } from '../types';

export default function RepoDetailPage() {
  const { id } = useParams();
  const [trends, setTrends] = useState<TrendPoint[]>([]);
  const [findings, setFindings] = useState<Finding[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    Promise.all([
      fetch(`/v1/repos/${id}/trends`, { credentials: 'include' }).then(res => res.json()),
      fetch(`/v1/repos/${id}/findings`, { credentials: 'include' }).then(res => res.json())
    ]).then(([trendData, findingData]) => {
      setTrends(trendData);
      setFindings(findingData);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [id]);

  const downloadCSV = () => {
    if (findings.length === 0) return;
    const headers = ["Severity", "File", "Line", "Algorithm", "Category", "Suggested Replacement"];
    const rows = findings.map(f => [f.severity, f.file, f.line, f.algorithm, f.category, f.suggested_replacement]);
    const csvContent = [headers.join(","), ...rows.map(r => r.map(cell => `"${cell}"`).join(","))].join("\n");
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", "vyala_cbom_export.csv");
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  if (loading) return <DashboardLayout><div>Loading repo details...</div></DashboardLayout>;

  return (
    <DashboardLayout>
      <Link to="/dashboard" className="text-blue-600 mb-4 inline-block">&larr; Back to Repositories</Link>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold">Repository Posture</h1>
        <button onClick={downloadCSV} className="bg-blue-600 text-white px-4 py-2 rounded shadow hover:bg-blue-700 transition-colors">
          Export CSV
        </button>
      </div>

      <div className="bg-white p-4 rounded shadow border mb-8">
        <h2 className="text-xl font-semibold mb-4">Open Findings (Last 90 Days)</h2>
        <div className="h-72 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={trends}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="high" stroke="#dc2626" strokeWidth={2} />
              <Line type="monotone" dataKey="medium" stroke="#f59e0b" strokeWidth={2} />
              <Line type="monotone" dataKey="low" stroke="#3b82f6" strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="bg-white p-4 rounded shadow border">
        <h2 className="text-xl font-semibold mb-4">Current Open Findings ({findings.length})</h2>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead className="bg-gray-50 border-b">
              <tr>
                <th className="px-4 py-2 text-left">Severity</th>
                <th className="px-4 py-2 text-left">File</th>
                <th className="px-4 py-2 text-left">Algorithm</th>
                <th className="px-4 py-2 text-left">Category</th>
                <th className="px-4 py-2 text-left">Suggested Replacement</th>
              </tr>
            </thead>
            <tbody>
              {findings.map(f => (
                <tr key={f.id} className="border-b">
                  <td className="px-4 py-2 font-medium">
                    <span className={f.severity === 'high' ? 'text-red-600' : f.severity === 'medium' ? 'text-yellow-600' : 'text-blue-600'}>
                      {f.severity}
                    </span>
                  </td>
                  <td className="px-4 py-2 font-mono">{f.file}:{f.line}</td>
                  <td className="px-4 py-2">{f.algorithm}</td>
                  <td className="px-4 py-2">{f.category}</td>
                  <td className="px-4 py-2 text-gray-600">{f.suggested_replacement}</td>
                </tr>
              ))}
              {findings.length === 0 && (
                <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No open vulnerabilities! Nice work.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </DashboardLayout>
  );
}