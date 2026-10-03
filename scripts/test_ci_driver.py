import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("ci_driver", Path(__file__).with_name("ci-driver.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class DriverShardTest(unittest.TestCase):
    def test_linux_costs_do_not_charge_host_only_work(self):
        names = ["TestHostOnly", "TestSlow", "TestOther"]
        durations = {"driver/short/TestHostOnly": 500,
                     "driver/short/TestSlow": 100,
                     "driver/short/TestOther": 99,
                     "driver/linux/short/TestHostOnly": 0,
                     "driver/linux/short/TestSlow": 100,
                     "driver/linux/short/TestOther": 99}
        with patch.object(module.sys, "platform", "linux"):
            self.assertEqual(module.shards(names, durations, "short", 2),
                             [["TestSlow"], ["TestOther", "TestHostOnly"]])
        with patch.object(module.sys, "platform", "darwin"):
            self.assertEqual(module.shards(names, durations, "short", 2),
                             [["TestHostOnly"], ["TestSlow", "TestOther"]])

    def test_linux_full_measurements_precede_short_estimates(self):
        names = ["TestFull", "TestOther"]
        durations = {"driver/full/TestFull": 100,
                     "driver/linux/short/TestFull": 1,
                     "driver/short/TestOther": 50}
        with patch.object(module.sys, "platform", "linux"):
            self.assertEqual(module.shards(names, durations, "full", 2),
                             [["TestFull"], ["TestOther"]])
            durations["driver/linux/full/TestFull"] = 10
            self.assertEqual(module.shards(names, durations, "full", 2),
                             [["TestOther"], ["TestFull"]])

    def test_full_mode_uses_short_estimates_without_full_measurements(self):
        names = ["TestSlow", "TestFast", "TestNew", "Example", "FuzzInput"]
        durations = {"driver/short/TestSlow": 100}
        self.assertEqual(module.shards(names, durations, "full", 3),
                         module.shards(names, durations, "short", 3))

    def test_all_tests_have_one_shard_in_both_modes(self):
        names = ["TestSlow", "TestFast", "TestNew", "Example", "FuzzInput"]
        durations = {"driver/short/TestSlow": 100, "driver/full/TestFast": 100}
        for mode in ["short", "full"]:
            for count in [1, 3, 8]:
                with self.subTest(mode=mode, count=count):
                    shards = module.shards(names, durations, mode, count)
                    self.assertCountEqual([name for shard in shards for name in shard], names)
                    self.assertEqual(shards, module.shards(list(reversed(names)), durations, mode, count))
                    slow = "TestSlow" if mode == "short" else "TestFast"
                    if count > 1:
                        self.assertEqual(shards[0], [slow])


if __name__ == "__main__":
    unittest.main()
