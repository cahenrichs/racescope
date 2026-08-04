import { createBrowserRouter, type RouteObject } from 'react-router-dom'
import App from './App'
import RacePage from './pages/RacePage'

export const routes: RouteObject[] = [
  { path: '/', element: <App /> },
  { path: '/races/:meetingID', element: <RacePage /> },
]

export const router = createBrowserRouter(routes)
