<script>
	import { appSettings } from '../store.js';
</script>

<div class="panel">
	<h2>⚙️ Configuración del Motor</h2>

	<div class="setting-group highlight">
		<h3>🖥️ Rendimiento y Hardware</h3>
		<label>
			Hilos de CPU concurrentes: <strong>{$appSettings.threads}</strong>
			<input type="range" min="1" max="16" bind:value={$appSettings.threads} class="slider" />
			<span class="note">A mayor cantidad, más rápido terminará. Se recomienda usar la cantidad de núcleos físicos de su procesador.</span>
		</label>
	</div>

	<div class="setting-group">
		<h3>🎵 Audio (Opus)</h3>
		<label>
			Bitrate (VBR Dinámico):
			<select bind:value={$appSettings.bitrate}>
				<option value="96k">96 kbps (Ahorro extremo de espacio)</option>
				<option value="128k">128 kbps (Recomendado, calidad CD)</option>
				<option value="160k">160 kbps (Transparente, para puristas)</option>
			</select>
		</label>
		
		<label>
			Nivel de Compresión Opus: <strong>{$appSettings.compressionLevel}</strong>
			<input type="range" min="0" max="10" bind:value={$appSettings.compressionLevel} class="slider" />
			<span class="note">10 = Máxima calidad y menor peso (Proceso lento). 0 = Proceso muy rápido (Menos eficiente en peso).</span>
		</label>
	</div>

	<div class="setting-group">
		<h3>🖼️ Portadas (Metadatos)</h3>
		<label>
			Resolución (Redimensionado Proporcional):
			<select bind:value={$appSettings.coverSize}>
				<option value={500}>500x500 px</option>
				<option value={700}>700x700 px</option>
				<option value={1000}>1000x1000 px</option>
			</select>
		</label>
		<p class="note">La app buscará portadas incrustadas o archivos cover.jpg/png locales y los adaptará automáticamente.</p>
	</div>
</div>

<style>
	.panel { padding: 2rem; max-width: 800px; margin: auto; }
	h2 { color: #00ffcc; margin-bottom: 2rem; }
	.setting-group { background: #1a1a1a; padding: 1.5rem; border-radius: 8px; margin-bottom: 1.5rem; border: 1px solid #333; }
	.highlight { border-color: #00ffcc; background: #112222; }
	h3 { margin-top: 0; color: #fff; font-size: 1.1rem; }
	label { display: flex; flex-direction: column; gap: 0.5rem; color: #ddd; margin-bottom: 1rem; }
	select { background: #333; color: white; border: 1px solid #555; padding: 0.5rem; border-radius: 4px; font-family: inherit; font-size: 1rem; }
	.slider { width: 100%; margin-top: 0.5rem; cursor: pointer; }
	.note { color: #888; font-size: 0.85rem; font-style: italic; }
	strong { color: #00ffcc; }
</style>