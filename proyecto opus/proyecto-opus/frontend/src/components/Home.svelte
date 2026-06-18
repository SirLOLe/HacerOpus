<script>
	import { SelectDirectory, ConvertLibrary } from '../../wailsjs/go/main/App.js';
	import { EventsOn } from '../../wailsjs/runtime/runtime.js';
	import { appSettings } from '../store.js';

	let inputDir = "";
	let outputDir = "";

	let status = {
		currentFile: "Esperando...",
		progress: 0,
		processed: 0,
		total: 0,
		errors: 0
	};

	let isProcessing = false;
	let resultMessage = "";

	async function chooseInput() {
		inputDir = await SelectDirectory("Seleccione la carpeta raíz con su música original");
	}

	async function chooseOutput() {
		outputDir = await SelectDirectory("Seleccione dónde guardar la nueva biblioteca Opus");
	}

	async function startEngine() {
		if (!inputDir || !outputDir) {
			resultMessage = "Por favor, defina ambas carpetas.";
			return;
		}

		isProcessing = true;
		resultMessage = `Escaneando biblioteca e iniciando transcodificación con ${$appSettings.threads} hilos...`;

		EventsOn("conversion_progress", (update) => {
			status = update;
		});

		let result = await ConvertLibrary(inputDir, outputDir, $appSettings);

		resultMessage = result;
		status.currentFile = "Finalizado";
		isProcessing = false;
	}
</script>

<div class="panel">
	<h2>🔄 Transcodificador Opus</h2>

	<div class="routes">
		<div class="route-item">
			<button class="btn" on:click={chooseInput}>Raíz Origen</button>
			<span>{inputDir || "Ninguna seleccionada"}</span>
		</div>
		<div class="route-item">
			<button class="btn" on:click={chooseOutput}>Raíz Destino</button>
			<span>{outputDir || "Ninguna seleccionada"}</span>
		</div>
	</div>

	<button class="btn-main" on:click={startEngine} disabled={isProcessing}>
		{isProcessing ? "Procesando Biblioteca..." : "INICIAR CONVERSIÓN MASIVA"}
	</button>

	{#if resultMessage}
		<p class="msg">{resultMessage}</p>
	{/if}

	<div class="tracker">
		<div class="stats">
			<p><strong>Archivo:</strong> {status.currentFile}</p>
			<p><strong>Procesados:</strong> {status.processed} / {status.total}</p>
			<p class="err"><strong>Errores:</strong> {status.errors}</p>
		</div>

		<div class="bar-bg">
			<div class="bar-fill" style="width: {status.progress}%"></div>
		</div>
		<p class="percent">{status.progress.toFixed(1)}%</p>
	</div>
</div>

<style>
	.panel { padding: 2rem; max-width: 800px; margin: auto; }
	h2 { color: #00ffcc; margin-bottom: 1.5rem; }
	.routes, .tracker { background: #1a1a1a; padding: 1.5rem; border-radius: 8px; margin-bottom: 1.5rem; border: 1px solid #333; }
	.route-item { display: flex; gap: 1rem; align-items: center; margin-bottom: 1rem; color: #999; font-family: monospace; word-break: break-all; }
	.btn { background: #444; color: white; border: none; padding: 0.6rem 1.2rem; cursor: pointer; border-radius: 4px; font-weight: bold; }
	.btn:hover { background: #555; }
	.btn-main { background: #00ffcc; color: #000; width: 100%; padding: 1rem; font-weight: bold; font-size: 1.1rem; border: none; cursor: pointer; border-radius: 4px; letter-spacing: 1px; }
	.btn-main:disabled { opacity: 0.5; cursor: not-allowed; }
	.stats { display: flex; justify-content: space-between; font-size: 0.95rem; color: #ddd; margin-bottom: 1rem; }
	.err { color: #ff4444; }
	.bar-bg { width: 100%; height: 12px; background: #333; border-radius: 6px; overflow: hidden; }
	.bar-fill { height: 100%; background: #00ffcc; transition: width 0.3s ease; }
	.percent { text-align: right; color: #00ffcc; font-family: monospace; margin-top: 0.5rem; font-size: 0.9rem; }
	.msg { color: #00ffcc; text-align: center; margin-bottom: 1rem; font-weight: bold; }
</style>