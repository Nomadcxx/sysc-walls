import os
import tempfile
import unittest
from pathlib import Path

import tour
import webp_report


class TourTest(unittest.TestCase):
    def test_timeline(self):
        scenes = tour.parse_timeline(Path(__file__).with_name("tour.timeline").read_text())
        self.assertTrue(200 <= sum(seconds for seconds, _ in scenes) <= 240)
        self.assertEqual([args[0] for _, args in scenes], [
            "ak-47-text", "matrix-art", "fire-text", "rain-art", "napoleon-bust",
            "higgs-field", "sysc-logo", "sonar", "lowe", "jason", "wasteland",
            "f-22-raptor", "f-14-tomcat", "roman-gladiator"])
        holds = {args[0]: seconds for seconds, args in scenes}
        self.assertGreaterEqual(holds["rain-art"], 60)
        self.assertGreaterEqual(holds["napoleon-bust"], 20)
        for effect in ("lowe", "jason", "f-22-raptor", "f-14-tomcat", "roman-gladiator"):
            self.assertGreaterEqual(holds[effect], 12)
        self.assertIn("higgs-field", [args[0] for _, args in scenes])
        self.assertIn("napoleon-bust", [args[0] for _, args in scenes])
        for bad in ("", "nan fire nord", "inf fire nord", "0 fire nord", "-1 fire nord", "1 fire"):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                tour.parse_timeline(bad)

    def test_children_finish_and_early_exit_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            script = Path(directory) / "display"
            pid = Path(directory) / "pid"
            script.write_text(f'#!/bin/sh\necho $$ > "{pid}"\nexec sleep 60\n')
            script.chmod(0o755)
            marks = {}
            tour.play([(0.03, ["fire", "nord"])], str(script), Path("art"), marks)
            self.assertGreater(marks["end"], marks["start"])
            with self.assertRaises(ProcessLookupError):
                os.kill(int(pid.read_text()), 0)
            with self.assertRaises(RuntimeError):
                tour.play([(0.03, ["fire", "nord"])], "/bin/false", Path("art"), {})

    def test_report_parses_animation(self):
        info = webp_report.parse_webpmux_info(
            "Canvas size: 1920 x 1080\nFeatures present: animation\n"
            "Background color : 0xFFFFFFFF  Loop Count : 0\nNumber of frames: 1\n"
            "No.: width height alpha x_offset y_offset duration dispose blend image_size compression\n"
            "1: 1920 1080 no 0 0 1000 none no 20 lossy\n", 100)
        self.assertEqual((info.width, info.height, info.duration_ms, info.loop_count), (1920, 1080, 1000, 0))


if __name__ == "__main__":
    unittest.main()
