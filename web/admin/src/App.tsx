import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AdminLayout } from './layouts/AdminLayout'
import { AuthGuard } from './layouts/AuthGuard'
import DashboardPage from './pages/dashboard'
import LoginPage from './pages/login'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <AuthGuard>
              {(profile) => (
                <AdminLayout profile={profile}>
                  <DashboardPage profile={profile} />
                </AdminLayout>
              )}
            </AuthGuard>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
