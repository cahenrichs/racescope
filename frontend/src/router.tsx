import { createBrowserRouter } from 'react-router-dom'
import App from './App'
import RacePage from './pages/RacePage'

export const router = createBrowserRouter([
  { path: '/', element: <App /> },
  { path: '/races/:meetingID', element: <RacePage /> },
])
