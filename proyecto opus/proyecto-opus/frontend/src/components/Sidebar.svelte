<script>
  import { createEventDispatcher } from 'svelte';
  
  // Exponemos la página actual para saber qué botón pintar como "activo"
  export let activePage = 'home';
  
  const dispatch = createEventDispatcher();

  // Función para avisar a App.svelte que queremos cambiar de página
  function navigate(page) {
    dispatch('changePage', page);
  }
</script>

<aside class="sidebar">
  <div class="logo">🎵 Normalizer Pro</div>
  
  <nav>
    <button 
      class:active={activePage === 'home'} 
      on:click={() => navigate('home')}>
      Inicio
    </button>
    
    <button 
      class:active={activePage === 'settings'} 
      on:click={() => navigate('settings')}>
      Ajustes
    </button>
  </nav>
</aside>

<style>
  .sidebar {
    width: 250px;
    background-color: #1a1a1a;
    border-right: 1px solid #333;
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .logo {
    padding: 20px;
    font-size: 1.2rem;
    font-weight: bold;
    color: #4ade80;
    text-align: center;
    border-bottom: 1px solid #333;
  }

  nav {
    display: flex;
    flex-direction: column;
    padding: 10px;
    gap: 5px;
  }

  button {
    background: transparent;
    border: none;
    color: #a0a0a0;
    padding: 12px 15px;
    text-align: left;
    border-radius: 6px;
    cursor: pointer;
    font-size: 1rem;
    transition: 0.2s;
  }

  button:hover {
    background-color: #2a2a2a;
    color: white;
  }

  button.active {
    background-color: #4ade80;
    color: black;
    font-weight: bold;
  }
</style>