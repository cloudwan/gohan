#!/usr/bin/env python3
import subprocess
import unittest

class Completion():
    def prepare(self, program, command):
        self.program=program
        self.COMP_LINE="%s %s" % (program, command)
        self.COMP_WORDS=self.COMP_LINE.rstrip()

        args=command.split()
        self.COMP_CWORD=len(args)
        self.COMP_POINT=len(self.COMP_LINE)
        if (self.COMP_LINE[-1] == ' '):
            self.COMP_WORDS += " " 
            self.COMP_CWORD += 1

    def run(self, completion_file, program, command):
        self.prepare(program, command)
        full_cmdline=r'source {compfile}; COMP_LINE="{COMP_LINE}" COMP_WORDS=({COMP_WORDS}) COMP_CWORD={COMP_CWORD} COMP_POINT={COMP_POINT}; $(complete -p {program} | sed "s/.*-F \\([^ ]*\\) .*/\\1/") && echo ${{COMPREPLY[*]}}'.format(
                compfile=completion_file, COMP_LINE=self.COMP_LINE, COMP_WORDS=self.COMP_WORDS, COMP_POINT=self.COMP_POINT, program=self.program, COMP_CWORD=self.COMP_CWORD
        )

        out = subprocess.Popen(['bash', '-i', '-c', full_cmdline], stdout=subprocess.PIPE)
        return out.communicate()

class CompletionTestCase(unittest.TestCase):

    def assertEqualCompletion(self, program, cline, line, words, cword, point):
        c = Completion()
        c.prepare(program, cline)
        self.assertEqual(c.program, program)
        self.assertEqual(c.COMP_LINE, line)
        self.assertEqual(c.COMP_WORDS, words)
        self.assertEqual(c.COMP_CWORD, cword)
        self.assertEqual(c.COMP_POINT, point)

class BashCompletionTest(unittest.TestCase):

    def run_complete(self, completion_file, program, command, expected):
        stdout,stderr = Completion().run(completion_file, program, command)
        self.assertEqual(stdout.decode("utf-8"), expected + '\n')

class GohanClientTestCases(BashCompletionTest):
    def test_1(self):
        self.run_complete("client p", "policy pet")
    def test_2(self):
        self.run_complete("client ", "version namespace event extension policy schema pet order")
    def test_3(self):
        self.run_complete("client pet","pet")
    def test_4(self):
        self.run_complete("client pet ","list show create set delete")
    def test_5(self):
        self.run_complete("client pet s","show set")
    def test_6(self):
        self.run_complete("client pet show -","--output-format --verbosity --fields --id --name --tenant_id --description --status")
    def test_7(self):
        self.run_complete("client pet show --fields ","id name tenant_id description status")
    def test_8(self):
        self.run_complete("client namespace show --fields ","id name description prefix parent version metadata")
    def test_9(self):
        self.run_complete("client namespace show --verbosity ","0 1 2")
    def test_10(self):
        self.run_complete("client policy show --fields ","id principal resource action effect condition")
    def test_11(self):
        self.run_complete("client policy something",""  )

    def run_complete(self, command, expected):
        completion_file="gohan_client_bash_completion.sh"
        program="gohan"
        super(GohanClientTestCases, self).run_complete(completion_file, program, command, expected)

if (__name__=='__main__'):
    unittest.main()
