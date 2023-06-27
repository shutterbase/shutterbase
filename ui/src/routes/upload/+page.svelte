<script lang="ts">
  import Dropzone from "dropzone";
  import { onMount } from 'svelte';
  import "dropzone/dist/dropzone.css";

  onMount(() => {
    let myDropzone = new Dropzone("#my-form");
    myDropzone.on("addedfile", (file: any) => {
      console.log(`File added: ${file.name}`);
    });
  });

</script>

<form action="/target" class="dropzone" id="my-form"></form>
