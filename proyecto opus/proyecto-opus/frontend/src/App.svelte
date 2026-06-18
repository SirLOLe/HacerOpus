<script>
	import Sidebar from './components/Sidebar.svelte';
	import Home from './components/Home.svelte';
	import Settings from './components/Settings.svelte';

	let currentPage = 'home';

	function handleNavigation(event) {
		currentPage = event.detail;
	}
</script>

<main class="app-container">
	<Sidebar activePage={currentPage} on:changePage={handleNavigation} />

	<section class="content-area">
		{#if currentPage === 'home'}
			<Home />
		{:else if currentPage === 'settings'}
			<Settings />
		{/if}
	</section>
</main>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
		background-color: #121212;
		font-family: system-ui, -apple-system, sans-serif;
	}

	.app-container {
		display: flex;
		height: 100vh;
		width: 100vw;
		overflow: hidden;
	}

	.content-area {
		flex-grow: 1;
		background-color: #121212;
		overflow-y: auto;
	}
</style>