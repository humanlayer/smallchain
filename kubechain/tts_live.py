#!/usr/bin/env python3
import os
import sys
import sounddevice as sd
import numpy as np
from openai import OpenAI
import io
from pydub import AudioSegment

# Initialize OpenAI client using environment variable
client = OpenAI()

def text_to_speech_stream(text, voice="alloy"):
    """Convert text to speech using OpenAI's API and stream it"""
    try:
        response = client.audio.speech.create(
            model="tts-1",
            voice=voice,
            input=text
        )
        
        # Convert response to audio data
        audio_data = io.BytesIO(response.content)
        audio = AudioSegment.from_mp3(audio_data)
        
        # Convert to numpy array for sounddevice
        samples = np.array(audio.get_array_of_samples()).astype(np.float32) / 32768.0
        if audio.channels == 2:
            samples = samples.reshape(-1, 2)
        
        # Play audio
        sd.play(samples, audio.frame_rate)
        sd.wait()  # Wait until audio is finished playing
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    # Read text from stdin
    text = sys.stdin.read().strip()
    if not text:
        print("Error: No input text provided", file=sys.stderr)
        sys.exit(1)
    
    text_to_speech_stream(text)