import '@fontsource-variable/newsreader/opsz.css'
import '@fontsource-variable/public-sans'
import './app.css'
import { mount } from 'svelte'
import App from './App.svelte'

export default mount(App, { target: document.getElementById('app') })
