import { createRoot } from 'react-dom/client';
import '../vendor/glacius-design-system/glacius.css';
import './app.css';
import './model.css';
import './chat.css';
import { App } from './App';

createRoot(document.getElementById('root')!).render(<App />);
