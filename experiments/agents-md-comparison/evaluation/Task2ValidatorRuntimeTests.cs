using System.Reflection;
using CompanyName.MyMeetings.Modules.Meetings.Application.Meetings.AddMeetingAttendee;
using CompanyName.MyMeetings.Modules.Meetings.Application.Meetings.CancelMeeting;
using FluentValidation;
using NUnit.Framework;

namespace Markitect.ComparisonEvaluation;

[TestFixture]
public sealed class Task2ValidatorRuntimeTests
{
    private static readonly Assembly ApplicationAssembly = typeof(AddMeetingAttendeeCommand).Assembly;

    [Test]
    public void AddMeetingAttendeeValidator_RejectsEmptyMeetingId_AndAcceptsNonEmptyMeetingId()
    {
        AssertGuidValidation(
            "CompanyName.MyMeetings.Modules.Meetings.Application.Meetings.AddMeetingAttendee.AddMeetingAttendeeCommandValidator",
            typeof(AddMeetingAttendeeCommand),
            new object[] { Guid.Empty, 0 },
            new object[] { new Guid("11111111-1111-1111-1111-111111111111"), 0 });
    }

    [Test]
    public void CancelMeetingValidator_RejectsEmptyMeetingId_AndAcceptsNonEmptyMeetingId()
    {
        AssertGuidValidation(
            "CompanyName.MyMeetings.Modules.Meetings.Application.Meetings.CancelMeeting.CancelMeetingCommandValidator",
            typeof(CancelMeetingCommand),
            new object[] { Guid.Empty },
            new object[] { new Guid("11111111-1111-1111-1111-111111111111") });
    }

    private static void AssertGuidValidation(
        string validatorTypeName,
        Type commandType,
        object[] invalidArguments,
        object[] validArguments)
    {
        var validatorType = ApplicationAssembly.GetType(validatorTypeName, throwOnError: false);
        Assert.That(validatorType, Is.Not.Null, $"Expected exact validator type {validatorTypeName}");
        Assert.That(validatorType!.IsNotPublic, Is.True, "Validator must remain nonpublic");

        var validatorContract = typeof(IValidator<>).MakeGenericType(commandType);
        Assert.That(validatorContract.IsAssignableFrom(validatorType), Is.True);
        var validator = Activator.CreateInstance(validatorType, nonPublic: true);
        Assert.That(validator, Is.Not.Null);

        var commandConstructor = commandType.GetConstructor(invalidArguments.Select(x => x.GetType()).ToArray());
        Assert.That(commandConstructor, Is.Not.Null, "Expected upstream command constructor signature");
        var validate = validatorContract.GetMethod("Validate", new[] { commandType });
        Assert.That(validate, Is.Not.Null);

        var invalidCommand = commandConstructor!.Invoke(invalidArguments);
        var invalidResult = validate!.Invoke(validator, new[] { invalidCommand });
        Assert.That(GetErrorCount(invalidResult), Is.GreaterThan(0), "Empty MeetingId should fail validation");

        var validCommand = commandConstructor.Invoke(validArguments);
        var validResult = validate.Invoke(validator, new[] { validCommand });
        Assert.That(GetErrorCount(validResult), Is.Zero, "Non-empty MeetingId should pass validation");
    }

    private static int GetErrorCount(object? validationResult)
    {
        Assert.That(validationResult, Is.Not.Null);
        var errors = validationResult!.GetType().GetProperty("Errors")?.GetValue(validationResult) as System.Collections.ICollection;
        Assert.That(errors, Is.Not.Null);
        return errors!.Count;
    }
}
